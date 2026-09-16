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
// above domain.MaxDxfFileBytes by that margin, and the use case enforces the
// real per-file limit on the decoded part.
const maxDxfUploadBytes = domain.MaxDxfFileBytes + (1 << 20)

// dxfFormFile is the multipart field the DXF is read from.
const dxfFormFile = "archivo"

type StoreLoteoDxfHandler struct {
	storeLoteoDxf loteos.StoreLoteoDxf
}

func NewStoreLoteoDxfHandler(storeLoteoDxf loteos.StoreLoteoDxf) *StoreLoteoDxfHandler {
	return &StoreLoteoDxfHandler{storeLoteoDxf: storeLoteoDxf}
}

// Handle stores the original DXF of an existing loteo. It must run behind
// middleware.RequireAuth.
func (handler *StoreLoteoDxfHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())

	request.Body = http.MaxBytesReader(w, request.Body, maxDxfUploadBytes)

	if err := request.ParseMultipartForm(1 << 20); err != nil {
		return domain.ErrInvalidDxfFile.WithCause(err)
	}

	file, header, err := request.FormFile(dxfFormFile)
	if err != nil {
		return domain.ErrInvalidDxfFile.WithCause(err)
	}
	defer file.Close()

	actor := loteos.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles}

	stored, err := handler.storeLoteoDxf.Execute(request.Context(), actor, loteos.StoreLoteoDxfInput{
		LoteoID:  request.PathValue("loteoId"),
		FileName: header.Filename,
		Content:  file,
		Size:     header.Size,
	})
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusCreated, dto.StoreLoteoDxfResponse{
		ID:           stored.ID,
		OriginalName: stored.OriginalName,
		MimeType:     stored.MimeType,
		Sha256:       stored.Sha256,
		CreatedAt:    stored.CreatedAt,
	})
	return nil
}
