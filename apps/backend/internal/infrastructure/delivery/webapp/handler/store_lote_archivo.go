package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/loteos"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type StoreLoteFileHandler struct {
	storeLoteFile loteos.StoreLoteFile
}

func NewStoreLoteFileHandler(storeLoteFile loteos.StoreLoteFile) *StoreLoteFileHandler {
	return &StoreLoteFileHandler{storeLoteFile: storeLoteFile}
}

// Handle attaches a new foto/plano to one lote. It must run behind
// middleware.RequireAuth.
func (handler *StoreLoteFileHandler) Handle(w http.ResponseWriter, request *http.Request) error {
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

	stored, err := handler.storeLoteFile.Execute(request.Context(), actor, loteos.StoreLoteFileInput{
		LoteoID:  request.PathValue("loteoId"),
		LoteID:   request.PathValue("loteId"),
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
