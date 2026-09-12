package domain

import (
	"bytes"
	"time"
)

// Categories a foto/plano upload may declare. "dxf" and "documento_legal" are
// recorded by other flows (StoreLoteoDxf, and the future escribano module)
// and are never accepted here.
const (
	FileCategoryFoto  = "foto"
	FileCategoryPlano = "plano"
)

// MaxFileBytes bounds one foto/plano upload. The frontend has no pre-upload
// size check of its own to mirror; this is the only enforcement.
const MaxFileBytes = 10_000_000

// MaxFilesPerEntity bounds how many active fotos/planos a single loteo or
// lote may carry, so an entity can't accumulate uploads without limit.
const MaxFilesPerEntity = 20

// fileMimeTypes are the only content types StoreLoteoFile/StoreLoteFile
// accept. Kept narrow (photos plus PDF for scanned planos) since anything
// stored here is later served back to a browser.
var fileMimeTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/webp":      true,
	"application/pdf": true,
}

func ValidFileCategory(category string) bool {
	return category == FileCategoryFoto || category == FileCategoryPlano
}

func ValidFileMimeType(mimeType string) bool {
	return fileMimeTypes[mimeType]
}

// jpegSignature, pngSignature, webpRiff, webpFormat and pdfSignature are the
// magic bytes a real file of each declared mimeType starts with. Checked
// against the upload's own content instead of trusting mimeType, which is
// only the client's Content-Type header and proves nothing about the bytes
// that follow it.
var (
	jpegSignature = []byte{0xFF, 0xD8, 0xFF}
	pngSignature  = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}
	webpRiff      = []byte("RIFF")
	webpFormat    = []byte("WEBP")
	pdfSignature  = []byte("%PDF-")
)

// FileContentMatchesMimeType reports whether header — the first bytes of an
// upload's content — carries the file signature expected for mimeType.
// header may be shorter than the signature it's checked against, in which
// case it never matches.
func FileContentMatchesMimeType(mimeType string, header []byte) bool {
	switch mimeType {
	case "image/jpeg":
		return bytes.HasPrefix(header, jpegSignature)
	case "image/png":
		return bytes.HasPrefix(header, pngSignature)
	case "image/webp":
		return len(header) >= 12 && bytes.Equal(header[0:4], webpRiff) && bytes.Equal(header[8:12], webpFormat)
	case "application/pdf":
		return bytes.HasPrefix(header, pdfSignature)
	default:
		return false
	}
}

var (
	ErrInvalidFile  = &Error{Kind: KindInvalid, Code: "invalid_archivo", Message: "El archivo es inválido, de un tipo no admitido o supera el tamaño permitido"}
	ErrTooManyFiles = &Error{Kind: KindInvalid, Code: "too_many_archivos", Message: "Se alcanzó el máximo de archivos permitidos"}
	ErrFileNotFound = &Error{Kind: KindNotFound, Code: "archivo_not_found", Message: "El archivo solicitado no existe"}
)

// NewFile is a foto or plano about to be recorded, once its bytes are stored
// in object storage under StorageKey.
type NewFile struct {
	Category     string
	StorageKey   string
	OriginalName string
	MimeType     string
	Sha256       string
}

// File is a recorded foto or plano attached to a loteo or one of its lotes.
type File struct {
	ID           string
	Category     string
	StorageKey   string
	OriginalName string
	MimeType     string
	Sha256       string
	CreatedAt    time.Time
}
