package http

import (
	"log/slog"
	"net/http"

	"github.com/purpose-robot/blips-and-chitz/auth"
	"github.com/purpose-robot/blips-and-chitz/health"
	httpauth "github.com/purpose-robot/blips-and-chitz/http/auth"
	httphealth "github.com/purpose-robot/blips-and-chitz/http/health"
)

type Server struct {
	logger        *slog.Logger
	limiter       LimiterConfig
	authService   *auth.Service
	healthService *health.Service
}

type LimiterConfig struct {
	Burst   int
	RPS     float64
	Enabled bool
}

func NewServer(logger *slog.Logger, limiter LimiterConfig, authService *auth.Service, healthService *health.Service) *Server {
	return &Server{
		logger:        logger,
		limiter:       limiter,
		authService:   authService,
		healthService: healthService,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	authHandler := httpauth.NewHandler(s.authService)
	healthHandler := httphealth.NewHandler(s.healthService)

	mux.Handle("POST /api/users", authHandler.RegisterUser())
	mux.Handle("PUT /api/users/activate", authHandler.ActivateUser())
	mux.Handle("PUT /api/users/password", authHandler.UpdateUserPassword())

	mux.Handle("POST /api/tokens/activation", authHandler.CreateActivationToken())
	mux.Handle("POST /api/tokens/password-reset", authHandler.CreatePasswordResetToken())
	mux.Handle("POST /api/tokens/authentication", authHandler.CreateAuthenticationToken())

	mux.HandleFunc("GET /api/health", authHandler.RequirePermission("health:read", healthHandler.Check))

	return s.logRequest(s.recoverPanic(s.rateLimit(authHandler.Authenticate(mux))))
}
