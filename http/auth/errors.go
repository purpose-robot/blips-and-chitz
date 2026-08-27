package auth

import (
	"errors"
	"net/http"

	"github.com/purpose-robot/blips-and-chitz/auth"
	httpx "github.com/purpose-robot/blips-and-chitz/internal/http"
)

func mapDomainError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, auth.ErrRecordNotFound):
		err = httpx.NewSafeError(err, http.StatusUnprocessableEntity, auth.ErrRecordNotFound.Error())

	case errors.Is(err, auth.ErrEditConflict):
		err = httpx.NewSafeError(err, http.StatusConflict, auth.ErrEditConflict.Error())

	case errors.Is(err, auth.ErrDuplicateEmail):
		err = httpx.NewSafeError(err, http.StatusConflict, auth.ErrDuplicateEmail.Error())

	case errors.Is(err, auth.ErrInvalidCredentials):
		err = httpx.NewSafeError(err, http.StatusUnauthorized, auth.ErrInvalidCredentials.Error())

	case errors.Is(err, auth.ErrAlreadyActivated):
		err = httpx.NewSafeError(err, http.StatusUnprocessableEntity, auth.ErrAlreadyActivated.Error())

	case errors.Is(err, auth.ErrInactiveAccount):
		err = httpx.NewSafeError(err, http.StatusForbidden, auth.ErrInactiveAccount.Error())

	case errors.Is(err, auth.ErrMissingPermission):
		err = httpx.NewSafeError(err, http.StatusForbidden, auth.ErrMissingPermission.Error())

	case errors.Is(err, auth.ErrAuthenticationRequired):
		err = httpx.NewSafeError(err, http.StatusUnauthorized, auth.ErrAuthenticationRequired.Error())

	case errors.Is(err, auth.ErrInvalidAuthenticationToken):
		err = httpx.NewSafeError(err, http.StatusUnauthorized, auth.ErrInvalidAuthenticationToken.Error())

	case errors.Is(err, auth.ErrInvalidActivationToken):
		err = httpx.NewSafeError(err, http.StatusUnprocessableEntity, auth.ErrInvalidActivationToken.Error())

	case errors.Is(err, auth.ErrInvalidPasswordResetToken):
		err = httpx.NewSafeError(err, http.StatusUnprocessableEntity, auth.ErrInvalidPasswordResetToken.Error())
	}

	httpx.Error(w, r, err)
}
