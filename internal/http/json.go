package http

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Envelope map[string]any

func WriteJSON(w http.ResponseWriter, status int, in any) error {
	return WriteJSONWithHeaders(w, status, in, nil)
}

func WriteJSONWithHeaders(w http.ResponseWriter, status int, in any, headers http.Header) error {
	b, err := json.Marshal(in, jsontext.WithIndent("\t"))
	if err != nil {
		return err
	}

	b = append(b, '\n')

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_, _ = w.Write(b)
	return nil
}

func ReadJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	return readJSON(w, r, destination, true)
}

func ReadJSONUnsafe(w http.ResponseWriter, r *http.Request, destination any) error {
	return readJSON(w, r, destination, false)
}

func readJSON(w http.ResponseWriter, r *http.Request, destination any, rejectUnknownMembers bool) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1_048_576)

	var opts []json.Options

	if rejectUnknownMembers {
		opts = append(opts, json.RejectUnknownMembers(true))
	}

	errorFunc := func(cause error, format string, args ...any) error {
		return NewSafeError(cause, http.StatusBadRequest, fmt.Sprintf(format, args...))
	}

	err := json.UnmarshalRead(r.Body, destination, opts...)
	if err != nil {
		var (
			maxBytesError  *http.MaxBytesError
			semanticError  *json.SemanticError
			syntacticError *jsontext.SyntacticError
		)

		switch {
		case errors.As(err, &syntacticError):
			if errors.Is(err, io.ErrUnexpectedEOF) && syntacticError.ByteOffset == 0 {
				return errorFunc(err, "body must not be empty")
			}

			return errorFunc(err, "body contains badly-formed JSON (at character %d)", syntacticError.ByteOffset)

		case errors.Is(err, json.ErrUnknownName) && errors.As(err, &semanticError):
			return errorFunc(err, "body contains unknown key %q", semanticError.JSONPointer.LastToken())

		case errors.As(err, &semanticError):
			if field := semanticError.JSONPointer; field != "" {
				return errorFunc(err, "body contains incorrect JSON type for field %q", strings.TrimPrefix(string(field), "/"))
			}

			return errorFunc(err, "body contains incorrect JSON type (at character %d)", semanticError.ByteOffset)

		case errors.As(err, &maxBytesError):
			return NewSafeError(err, http.StatusUnprocessableEntity, fmt.Sprintf("body must not be larger than %d bytes", maxBytesError.Limit))

		default:
			return errorFunc(err, "the request body could not be parsed")
		}
	}

	return nil
}
