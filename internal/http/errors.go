package http

import (
	"errors"
	"net/http"

	slogx "github.com/purpose-robot/blips-and-chitz/internal/slog"
	"github.com/purpose-robot/blips-and-chitz/internal/validator"
)

type SafeError struct {
	Cause   error             `json:"-"`
	Status  int               `json:"-"`
	UserMsg string            `json:"message"`
	Details map[string]string `json:"details,omitzero"`
}

func (e *SafeError) Error() string {
	return e.UserMsg
}

func NewSafeError(cause error, status int, message string) *SafeError {
	return NewSafeErrorDetails(cause, status, message, nil)
}

func NewSafeErrorDetails(cause error, status int, message string, details map[string]string) *SafeError {
	return &SafeError{
		Cause:   cause,
		Status:  status,
		UserMsg: message,
		Details: details,
	}
}

func Error(w http.ResponseWriter, r *http.Request, err error) {
	slogx.AddField(r.Context(), slogx.Error(err))

	if safeErr, ok := errors.AsType[*SafeError](err); ok {
		if safeErr.Cause != nil && safeErr.Status >= http.StatusInternalServerError {
			slogx.AddField(r.Context(), slogx.Cause(safeErr.Cause))
		}

		_ = WriteJSON(w, safeErr.Status, Envelope{"error": safeErr})
		return
	}

	if valErrs, ok := errors.AsType[validator.Errors](err); ok {
		_ = WriteJSON(w, http.StatusUnprocessableEntity, Envelope{"error": &SafeError{UserMsg: "invalid fields", Details: valErrs}})
		return
	}

	_ = WriteJSON(w, http.StatusInternalServerError, Envelope{"error": "the server encountered a problem and could not process your request"})
}
