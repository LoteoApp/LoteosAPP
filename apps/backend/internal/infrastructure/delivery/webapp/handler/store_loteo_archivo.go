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
// above domain.MaxFileBytes by that margin, and the use case enforces
// the real per-file limit on the decoded part.
const maxFileUploadBytes = domain.MaxFileBytes + (1 << 20)

// fileFormField and categoryFormField are the multipart fields a foto/plano
// upload is read from.
const (
	fileFormField     = "archivo"
	categoryFormField = "categoria"
)

type StoreLoteoFileHandler struct {
	storeLoteoFile loteos.StoreLoteoFile
}

func NewStoreLoteoFileHandler(storeLoteoFile loteos.StoreLoteoFile) *StoreLoteoFileHandler {
	return &StoreLoteoFileHandler{storeLoteoFile: storeLoteoFile}
}

// Handle attaches a new foto/plano to a loteo. It must run behind
// middleware.RequireAuth.
func (handler *StoreLoteoFileHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())

	request.Body = http.MaxBytesReader(w, request.Body, maxFileUploadBytes)

	if err := request.ParseMultipartForm(1 << 20); err != nil {
		return domain.ErrInvalidFile.WithCause(err)
	}

	file, header, err := request.FormFile(fileFormField)
	if err != nil {
		return domain.ErrInvalidFile.WithCause(err)
	}
	defer file.Close()

	actor := loteos.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles}

	stored, err := handler.storeLoteoFile.Execute(request.Context(), actor, loteos.StoreLoteoFileInput{
		LoteoID:  request.PathValue("loteoId"),
		Category: request.FormValue(categoryFormField),
		FileName: header.Filename,
		MimeType: header.Header.Get("Content-Type"),
		Content:  file,
		Size:     header.Size,
	})
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusCreated, dto.FileFromDomain(stored))
	return nil
}
