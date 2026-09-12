package dto

import (
	"time"

	"loteosapp/backend/internal/business/domain"
)

type FileResponse struct {
	ID           string    `json:"id"`
	Category     string    `json:"categoria"`
	OriginalName string    `json:"nombreOriginal"`
	MimeType     string    `json:"mimeType"`
	Sha256       string    `json:"hashSha256"`
	CreatedAt    time.Time `json:"fechaCreacion"`
}

type ListFilesResponse struct {
	Files []FileResponse `json:"archivos"`
}

func FileFromDomain(file domain.File) FileResponse {
	return FileResponse{
		ID:           file.ID,
		Category:     file.Category,
		OriginalName: file.OriginalName,
		MimeType:     file.MimeType,
		Sha256:       file.Sha256,
		CreatedAt:    file.CreatedAt,
	}
}

func FilesFromDomain(files []domain.File) []FileResponse {
	responses := make([]FileResponse, len(files))
	for index, file := range files {
		responses[index] = FileFromDomain(file)
	}

	return responses
}
