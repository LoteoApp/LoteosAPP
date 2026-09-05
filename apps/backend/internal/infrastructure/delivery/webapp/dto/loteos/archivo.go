package dto

import (
	"time"

	"loteosapp/backend/internal/business/domain"
)

type ArchivoResponse struct {
	ID           string    `json:"id"`
	Categoria    string    `json:"categoria"`
	OriginalName string    `json:"nombreOriginal"`
	MimeType     string    `json:"mimeType"`
	Sha256       string    `json:"hashSha256"`
	CreatedAt    time.Time `json:"fechaCreacion"`
}

type ListArchivosResponse struct {
	Archivos []ArchivoResponse `json:"archivos"`
}

func ArchivoFromDomain(archivo domain.Archivo) ArchivoResponse {
	return ArchivoResponse{
		ID:           archivo.ID,
		Categoria:    archivo.Categoria,
		OriginalName: archivo.OriginalName,
		MimeType:     archivo.MimeType,
		Sha256:       archivo.Sha256,
		CreatedAt:    archivo.CreatedAt,
	}
}

func ArchivosFromDomain(archivos []domain.Archivo) []ArchivoResponse {
	responses := make([]ArchivoResponse, len(archivos))
	for index, archivo := range archivos {
		responses[index] = ArchivoFromDomain(archivo)
	}

	return responses
}
