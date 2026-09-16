package handler

import (
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"

	"loteosapp/backend/internal/business/usecase/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

type GetFileContentHandler struct {
	getFileContent loteos.GetFileContent
}

func NewGetFileContentHandler(getFileContent loteos.GetFileContent) *GetFileContentHandler {
	return &GetFileContentHandler{getFileContent: getFileContent}
}

// Handle streams a foto/plano's own bytes. It must run behind
// middleware.RequireAuth. Unlike every other handler in this package it
// writes its success response directly, since the body isn't JSON.
func (handler *GetFileContentHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	actor := loteos.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles}

	content, err := handler.getFileContent.Execute(
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
	w.Header().Set("Content-Type", content.File.MimeType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{
		"filename": content.File.OriginalName,
	}))
	if content.Size > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(content.Size, 10))
	}
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, content.Body); err != nil {
		slog.ErrorContext(request.Context(), "streaming file content failed", "error", err)
		// The 200 and part of the body are already on the wire, so this
		// can't turn into a JSON error response. Aborting the connection
		// (rather than returning normally) keeps a truncated stream from
		// ever looking like a complete, successful download to the client.
		panic(http.ErrAbortHandler)
	}

	return nil
}
