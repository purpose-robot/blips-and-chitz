package health

import (
	"net/http"

	"github.com/purpose-robot/blips-and-chitz/health"
	httpx "github.com/purpose-robot/blips-and-chitz/internal/http"
	slogx "github.com/purpose-robot/blips-and-chitz/internal/slog"
)

type Handler struct {
	service *health.Service
}

func NewHandler(service *health.Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	response, err := h.service.Check(r.Context())
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	err = httpx.WriteJSON(w, http.StatusOK, httpx.Envelope{"response": response})
	if err != nil {
		slogx.AddField(r.Context(), slogx.Error(err))
	}
}
