package middleware

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"gotravel/internal/apperr"
	"gotravel/internal/logger"
	"gotravel/internal/response"
	"gotravel/internal/service"
)

type ctxKey int

const userKey ctxKey = 1

type User struct {
	ID       int64
	Username string
}

func UserFrom(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userKey).(User)
	return u, ok
}

func CORS(origin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(c int) {
	w.code = c
	w.ResponseWriter.WriteHeader(c)
}

func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, apperr.New(500, apperr.Internal, "hijacker not supported")
	}
	return h.Hijack()
}

func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func Access(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w, code: 200}
		start := time.Now()
		next.ServeHTTP(sw, r)
		logger.L.Info("http", "method", r.Method, "path", r.URL.Path, "status", sw.code, "ms", time.Since(start).Milliseconds())
	})
}

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.L.Error("panic", "err", rec)
				response.Fail(w, apperr.New(http.StatusInternalServerError, apperr.Internal, "panic"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func Auth(a *service.Auth) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if h == "" {
				h = r.URL.Query().Get("token")
			}
			h = strings.TrimPrefix(h, "Bearer ")
			if h == "" {
				response.Fail(w, apperr.New(http.StatusUnauthorized, apperr.Unauthorized, "missing token"))
				return
			}
			c, err := a.Parse(h)
			if err != nil {
				response.Fail(w, err)
				return
			}
			ctx := context.WithValue(r.Context(), userKey, User{ID: c.UserID, Username: c.Username})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
