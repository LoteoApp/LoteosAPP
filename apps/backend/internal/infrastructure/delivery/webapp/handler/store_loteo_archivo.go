package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/loteos"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

// The multipart envelope adds a little over the raw file; the body cap sits
// above domain.MaxArchivoFileBytes by that margin, and the use case enforces
// the real per-file limit on the decoded part.
const maxArchivoUploadBytes = domain.MaxArchivoFileBytes + (1 << 20)

// archivoFormFile and archivoFormCategoria are the multipart fields a
// foto/plano upload is read from.
const (
	archivoFormFile      = "archivo"
	archivoFormCategoria = "categoria"
)

type StoreLoteoArchivoHandler struct {
	storeLoteoArchivo loteos.StoreLoteoArchivo
}

func NewStoreLoteoArchivoHandler(storeLoteoArchivo loteos.StoreLoteoArchivo) *StoreLoteoArchivoHandler {
	return &StoreLoteoArchivoHandler{storeLoteoArchivo: storeLoteoArchivo}
}

// Handle attaches a new foto/plano to a loteo. It must run behind
// middleware.RequireAuth.
func (handler *StoreLoteoArchivoHandler) Handle(w http.ResponseWriter, request *http.Request) error {
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

	archivo, err := handler.storeLoteoArchivo.Execute(request.Context(), actor, loteos.StoreLoteoArchivoInput{
		LoteoID:   request.PathValue("loteoId"),
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
