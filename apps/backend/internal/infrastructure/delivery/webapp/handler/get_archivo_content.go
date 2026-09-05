package handler

import (
	"io"
	"log/slog"
	"mime"
	"net/http"

	"loteosapp/backend/internal/business/usecase/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

type GetArchivoContentHandler struct {
	getArchivoContent loteos.GetArchivoContent
}

func NewGetArchivoContentHandler(getArchivoContent loteos.GetArchivoContent) *GetArchivoContentHandler {
	return &GetArchivoContentHandler{getArchivoContent: getArchivoContent}
}

// Handle streams a foto/plano's own bytes. It must run behind
// middleware.RequireAuth. Unlike every other handler in this package it
// writes its success response directly, since the body isn't JSON.
func (handler *GetArchivoContentHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	actor := loteos.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles}

	content, err := handler.getArchivoContent.Execute(
		request.Context(), actor, request.PathValue("loteoId"), request.PathValue("archivoId"),
	)
	if err != nil {
		return err
	}
	defer content.Body.Close()

	// inline (not attachment) so a browser renders an image/PDF instead of
	// downloading it; the frontend fetches this with the caller's token and
	// turns the response into an object URL, so it never appears as a plain
	// <img src> that a browser could request unauthenticated.
	w.Header().Set("Content-Type", content.Archivo.MimeType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{
		"filename": content.Archivo.OriginalName,
	}))
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, content.Body); err != nil {
		slog.ErrorContext(request.Context(), "streaming archivo content failed", "error", err)
	}

	return nil
}
