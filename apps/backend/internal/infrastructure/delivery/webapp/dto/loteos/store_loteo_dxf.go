package dto

import "time"

type StoreLoteoDxfResponse struct {
	ID           string    `json:"id"`
	OriginalName string    `json:"nombreOriginal"`
	MimeType     string    `json:"mimeType"`
	Sha256       string    `json:"hashSha256"`
	CreatedAt    time.Time `json:"fechaCreacion"`
}
