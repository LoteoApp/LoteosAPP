package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type userService interface {
	CreateUser(ctx context.Context, actorRoles []string, email, rol string) (domain.Usuario, string, error)
	CompleteProfile(ctx context.Context, subject, nombre, apellido string) (domain.Usuario, error)
}

type UserHandler struct {
	service userService
}

func NewUserHandler(service userService) *UserHandler {
	return &UserHandler{service: service}
}

type createUserRequest struct {
	Email string `json:"email"`
	Rol   string `json:"rol"`
}

type createUserResponse struct {
	domain.Usuario
	TemporaryPassword string `json:"temporaryPassword"`
}

// Create handles the administrador-only user sign-up. It must run behind
// middleware.RequireAuth.
func (handler *UserHandler) Create(w http.ResponseWriter, request *http.Request) {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())

	var body createUserRequest
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		writeBadRequest(w, "invalid_body", "Cuerpo de la solicitud inválido")
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 5*time.Second)
	defer cancel()

	usuario, temporaryPassword, err := handler.service.CreateUser(ctx, principal.Roles, body.Email, body.Rol)
	if err != nil {
		writeUserError(w, request, "create user failed", err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, createUserResponse{
		Usuario:           usuario,
		TemporaryPassword: temporaryPassword,
	})
}

type completeProfileRequest struct {
	Nombre   string `json:"nombre"`
	Apellido string `json:"apellido"`
}

// CompleteProfile lets the authenticated caller fill in their own profile.
// It must run behind middleware.RequireAuth.
func (handler *UserHandler) CompleteProfile(w http.ResponseWriter, request *http.Request) {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())

	var body completeProfileRequest
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		writeBadRequest(w, "invalid_body", "Cuerpo de la solicitud inválido")
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 5*time.Second)
	defer cancel()

	usuario, err := handler.service.CompleteProfile(ctx, principal.Subject, body.Nombre, body.Apellido)
	if err != nil {
		writeUserError(w, request, "complete profile failed", err)
		return
	}

	response.WriteJSON(w, http.StatusOK, usuario)
}

func writeUserError(w http.ResponseWriter, request *http.Request, logMsg string, err error) {
	switch {
	case errors.Is(err, domain.ErrNoAutorizado):
		response.WriteJSON(w, http.StatusForbidden, response.ErrorResponse{
			Code: "forbidden", Message: "No tenés permisos para esta acción",
		})
	case errors.Is(err, domain.ErrEmailInvalido):
		writeBadRequest(w, "invalid_email", "Email inválido")
	case errors.Is(err, domain.ErrRolInvalido):
		writeBadRequest(w, "invalid_rol", "Rol inválido")
	case errors.Is(err, domain.ErrPerfilInvalido):
		writeBadRequest(w, "invalid_profile", "Nombre y apellido son obligatorios")
	case errors.Is(err, domain.ErrEmailEnUso):
		response.WriteJSON(w, http.StatusConflict, response.ErrorResponse{
			Code: "email_in_use", Message: "El email ya está en uso",
		})
	case errors.Is(err, domain.ErrUsuarioNoEncontrado):
		response.WriteJSON(w, http.StatusNotFound, response.ErrorResponse{
			Code: "user_not_found", Message: "Usuario no encontrado",
		})
	default:
		slog.ErrorContext(request.Context(), logMsg, "error", err)
		response.WriteJSON(w, http.StatusInternalServerError, response.ErrorResponse{
			Code: "internal_error", Message: "Ocurrió un error inesperado",
		})
	}
}

func writeBadRequest(w http.ResponseWriter, code, message string) {
	response.WriteJSON(w, http.StatusBadRequest, response.ErrorResponse{Code: code, Message: message})
}
