package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/purpose-robot/blips-and-chitz/auth"
)

func (h *Handler) RequirePermission(code string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, found := auth.ContextGetAuthenticatedUser(r.Context())
		if !found {
			w.Header().Set("WWW-Authenticate", "Bearer")

			mapDomainError(w, r, auth.ErrAuthenticationRequired)
			return
		}

		err := h.service.Authorize(r.Context(), &actor, code)
		if err != nil {
			mapDomainError(w, r, err)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func (h *Handler) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")

		authenticationHeader := r.Header.Get("Authorization")

		if authenticationHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		plaintext, ok := strings.CutPrefix(authenticationHeader, "Bearer ")
		if !ok {
			w.Header().Set("WWW-Authenticate", "Bearer")

			mapDomainError(w, r, auth.ErrInvalidAuthenticationToken)
			return
		}

		actor, err := h.service.Authenticate(r.Context(), auth.ScopeAuthentication, plaintext)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidAuthenticationToken) {
				w.Header().Set("WWW-Authenticate", "Bearer")
			}

			mapDomainError(w, r, err)
			return
		}

		next.ServeHTTP(w, r.WithContext(auth.ContextSetAuthenticatedUser(r.Context(), actor)))
	})
}
