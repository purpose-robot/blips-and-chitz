package http

import (
	"context"
	"net/http"

	slogx "github.com/purpose-robot/blips-and-chitz/internal/slog"
)

type ErrorFunc func(http.ResponseWriter, *http.Request, error)

type TargetFunc[In, Out any] func(context.Context, In) (Out, error)

func Handle[In, Out any](status int, key string, errorFunc ErrorFunc, targetFunc TargetFunc[In, Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input In

		err := ReadJSON(w, r, &input)
		if err != nil {
			Error(w, r, err)
			return
		}

		output, err := targetFunc(r.Context(), input)
		if err != nil {
			errorFunc(w, r, err)
			return
		}

		err = WriteJSON(w, status, Envelope{key: output})
		if err != nil {
			slogx.AddField(r.Context(), slogx.Error(err))
		}
	}
}
