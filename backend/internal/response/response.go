package response

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"

	"gotravel/internal/apperr"
	"gotravel/internal/logger"
)

var jsonScratch struct {
	sync.Mutex
	bytes.Buffer
}

type Envelope struct {
	OK    bool   `json:"ok"`
	Data  any    `json:"data,omitempty"`
	Error any    `json:"error,omitempty"`
	Code  string `json:"code,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	jsonScratch.Lock()
	jsonScratch.Reset()
	_ = json.NewEncoder(&jsonScratch.Buffer).Encode(Envelope{OK: status < 400, Data: data})
	payload := jsonScratch.Bytes()
	jsonScratch.Unlock()
	_, _ = w.Write(payload)
}

func Fail(w http.ResponseWriter, err error) {
	ae, ok := err.(*apperr.Error)
	if !ok {
		logger.L.Error("unhandled", "err", err)
		ae = apperr.New(http.StatusInternalServerError, apperr.Internal, "internal error")
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(ae.HTTP)
	_ = json.NewEncoder(w).Encode(Envelope{
		OK:    false,
		Code:  string(ae.Code),
		Error: map[string]any{"message": ae.Message, "detail": ae.Detail},
	})
}

func Decode(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return apperr.New(http.StatusBadRequest, apperr.BadRequest, "invalid json: "+err.Error())
	}
	return nil
}
