package auth

import (
	"context"
	"net/http"

	"github.com/purpose-robot/blips-and-chitz/auth"
	httpx "github.com/purpose-robot/blips-and-chitz/internal/http"
)

type Handler struct {
	service *auth.Service
}

func NewHandler(service *auth.Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) RegisterUser() http.HandlerFunc {
	type input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	return handle(http.StatusCreated, "response",
		func(ctx context.Context, actor *auth.User, in input) (*auth.User, error) {
			return h.service.RegisterUser(ctx, actor, auth.RegisterUser(in))
		})
}

func (h *Handler) ActivateUser() http.HandlerFunc {
	type input struct {
		Plaintext string `json:"plaintext"`
	}

	return handle(http.StatusOK, "response",
		func(ctx context.Context, actor *auth.User, in input) (*auth.User, error) {
			return h.service.ActivateUser(ctx, actor, auth.ActivateUser(in))
		})
}

func (h *Handler) UpdateUserPassword() http.HandlerFunc {
	type input struct {
		Password  string `json:"password"`
		Plaintext string `json:"plaintext"`
	}

	return handle(http.StatusOK, "message",
		func(ctx context.Context, actor *auth.User, in input) (string, error) {
			err := h.service.UpdateUserPassword(ctx, actor, auth.UpdateUserPassword(in))
			if err != nil {
				return "", err
			}

			return "your password was successfully reset", nil
		})
}

func (h *Handler) CreateActivationToken() http.HandlerFunc {
	type input struct {
		Email string `json:"email"`
	}

	return handle(http.StatusAccepted, "message",
		func(ctx context.Context, actor *auth.User, in input) (string, error) {
			err := h.service.CreateActivationToken(ctx, actor, auth.CreateActivationToken(in))
			if err != nil {
				return "", err
			}

			return "an email will be sent to you containing instructions to activate your account", nil
		})
}

func (h *Handler) CreatePasswordResetToken() http.HandlerFunc {
	type input struct {
		Email string `json:"email"`
	}

	return handle(http.StatusAccepted, "message",
		func(ctx context.Context, actor *auth.User, in input) (string, error) {
			err := h.service.CreatePasswordResetToken(ctx, actor, auth.CreatePasswordResetToken(in))
			if err != nil {
				return "", err
			}

			return "an email will be sent to you containing instructions to reset the password for your account", nil
		})
}

func (h *Handler) CreateAuthenticationToken() http.HandlerFunc {
	type input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	return handle(http.StatusCreated, "authentication_token",
		func(ctx context.Context, actor *auth.User, in input) (*auth.Token, error) {
			return h.service.CreateAuthenticationToken(ctx, actor, auth.CreateAuthenticationToken(in))
		})
}

func handle[In, Out any](status int, key string, targetFunc func(ctx context.Context, actor *auth.User, in In) (Out, error)) http.HandlerFunc {
	return httpx.Handle(status, key, mapDomainError,
		func(ctx context.Context, in In) (Out, error) {
			actor := new(auth.User)
			if user, ok := auth.ContextGetAuthenticatedUser(ctx); ok {
				actor = &user
			}

			return targetFunc(ctx, actor, in)
		})
}
