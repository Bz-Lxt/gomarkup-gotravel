package apperr

import "fmt"

type Code string

const (
	BadRequest     Code = "BAD_REQUEST"
	Unauthorized   Code = "UNAUTHORIZED"
	Forbidden      Code = "FORBIDDEN"
	NotFound       Code = "NOT_FOUND"
	Conflict       Code = "CONFLICT"
	RateLimited    Code = "RATE_LIMITED"
	Validation     Code = "VALIDATION"
	CycleDetected  Code = "CYCLE_DETECTED"
	Internal       Code = "INTERNAL"
	PayloadTooBig  Code = "PAYLOAD_TOO_LARGE"
	GeoUnavailable Code = "GEO_UNAVAILABLE"
)

type Error struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	Detail  any    `json:"detail,omitempty"`
	HTTP    int    `json:"-"`
}

func (e *Error) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

func New(http int, code Code, msg string) *Error {
	return &Error{HTTP: http, Code: code, Message: msg}
}

func WithDetail(http int, code Code, msg string, detail any) *Error {
	return &Error{HTTP: http, Code: code, Message: msg, Detail: detail}
}

func Is(err error, code Code) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == code
	}
	return false
}
