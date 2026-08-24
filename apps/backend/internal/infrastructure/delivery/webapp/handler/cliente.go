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

type clientService interface {
	List(ctx context.Context, actorRoles []string, buscar string) ([]domain.Cliente, error)
	Create(ctx context.Context, actorRoles []string, actorSubject string, cliente domain.Cliente) (domain.Cliente, error)
	Update(ctx context.Context, actorRoles []string, actorSubject string, cliente domain.Cliente) (domain.Cliente, error)
	Delete(ctx context.Context, actorRoles []string, actorSubject, id string) error
}

type ClientHandler struct {
	service clientService
}

func NewClientHandler(service clientService) *ClientHandler {
	return &ClientHandler{service: service}
}

type clienteRequest struct {
	Nombre   string `json:"nombre"`
	Apellido string `json:"apellido"`
	DNI      string `json:"dni"`
	Celular  string `json:"celular"`
	Email    string `json:"email"`
}

// List returns the clients the caller may see. It must run behind
// middleware.RequireAuth.
func (handler *ClientHandler) List(w http.ResponseWriter, request *http.Request) {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())

	ctx, cancel := context.WithTimeout(request.Context(), 5*time.Second)
	defer cancel()

	clientes, err := handler.service.List(ctx, principal.Roles, request.URL.Query().Get("buscar"))
	if err != nil {
		writeClientError(w, request, "list clients failed", err)
		return
	}

	response.WriteJSON(w, http.StatusOK, clientes)
}

// Create registers a new client. It must run behind middleware.RequireAuth.
func (handler *ClientHandler) Create(w http.ResponseWriter, request *http.Request) {
	principal, _ := middleware.PrincipalFromContext(request.Context())

	var body clienteRequest
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		writeBadRequest(w, "invalid_body", "Cuerpo de la solicitud inválido")
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 5*time.Second)
	defer cancel()

	cliente, err := handler.service.Create(ctx, principal.Roles, principal.Subject, body.toDomain(""))
	if err != nil {
		writeClientError(w, request, "create client failed", err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, cliente)
}

// Update edits an existing client. It must run behind middleware.RequireAuth.
func (handler *ClientHandler) Update(w http.ResponseWriter, request *http.Request) {
	principal, _ := middleware.PrincipalFromContext(request.Context())

	var body clienteRequest
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		writeBadRequest(w, "invalid_body", "Cuerpo de la solicitud inválido")
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), 5*time.Second)
	defer cancel()

	cliente, err := handler.service.Update(ctx, principal.Roles, principal.Subject, body.toDomain(request.PathValue("id")))
	if err != nil {
		writeClientError(w, request, "update client failed", err)
		return
	}

	response.WriteJSON(w, http.StatusOK, cliente)
}

// Delete gives the client the baja. It must run behind
// middleware.RequireAuth.
func (handler *ClientHandler) Delete(w http.ResponseWriter, request *http.Request) {
	principal, _ := middleware.PrincipalFromContext(request.Context())

	ctx, cancel := context.WithTimeout(request.Context(), 5*time.Second)
	defer cancel()

	if err := handler.service.Delete(ctx, principal.Roles, principal.Subject, request.PathValue("id")); err != nil {
		writeClientError(w, request, "delete client failed", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (body clienteRequest) toDomain(id string) domain.Cliente {
	return domain.Cliente{
		ID:       id,
		Nombre:   body.Nombre,
		Apellido: body.Apellido,
		DNI:      body.DNI,
		Celular:  body.Celular,
		Email:    body.Email,
	}
}

func writeClientError(w http.ResponseWriter, request *http.Request, logMsg string, err error) {
	switch {
	case errors.Is(err, domain.ErrNoAutorizado):
		response.WriteJSON(w, http.StatusForbidden, response.ErrorResponse{
			Code: "forbidden", Message: "No tenés permisos para esta acción",
		})
	case errors.Is(err, domain.ErrClienteInvalido):
		writeBadRequest(w, "invalid_client", "Nombre, apellido y DNI son obligatorios")
	case errors.Is(err, domain.ErrDNIEnUso):
		response.WriteJSON(w, http.StatusConflict, response.ErrorResponse{
			Code: "dni_in_use", Message: "Ya existe un cliente con ese DNI",
		})
	case errors.Is(err, domain.ErrClienteNoEncontrado):
		response.WriteJSON(w, http.StatusNotFound, response.ErrorResponse{
			Code: "client_not_found", Message: "Cliente no encontrado",
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
