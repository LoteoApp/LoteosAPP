package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/loteos"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type StoreLoteArchivoHandler struct {
	storeLoteArchivo loteos.StoreLoteArchivo
}

func NewStoreLoteArchivoHandler(storeLoteArchivo loteos.StoreLoteArchivo) *StoreLoteArchivoHandler {
	return &StoreLoteArchivoHandler{storeLoteArchivo: storeLoteArchivo}
}

// Handle attaches a new foto/plano to one lote. It must run behind
// middleware.RequireAuth.
func (handler *StoreLoteArchivoHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())

	request.Body = http.MaxBytesReader(w, request.Body, maxArchivoUploadBytes)

	if err := request.ParseMultipartForm(1 << 20); err != nil {
		return domain.ErrInvalidArchivo.WithCause(err)
	}

	file, header, err := request.FormFile(archivoFormFile)
	if err != nil {
		return domain.ErrInvalidArchivo.WithCause(err)
	}
	defer file.Close()

	actor := loteos.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles}

	archivo, err := handler.storeLoteArchivo.Execute(request.Context(), actor, loteos.StoreLoteArchivoInput{
		LoteoID:   request.PathValue("loteoId"),
		LoteID:    request.PathValue("loteId"),
		Categoria: request.FormValue(archivoFormCategoria),
		FileName:  header.Filename,
		MimeType:  header.Header.Get("Content-Type"),
		Content:   file,
		Size:      header.Size,
	})
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusCreated, dto.ArchivoFromDomain(archivo))
	return nil
}
