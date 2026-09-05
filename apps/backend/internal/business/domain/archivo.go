package domain

import "time"

// Categories a foto/plano upload may declare. "dxf" and "documento_legal" are
// recorded by other flows (StoreLoteoDxf, and the future escribano module)
// and are never accepted here.
const (
	ArchivoCategoriaFoto  = "foto"
	ArchivoCategoriaPlano = "plano"
)

// MaxArchivoFileBytes bounds one foto/plano upload. It mirrors
// MAX_ARCHIVO_FILE_BYTES in apps/frontend/src/features/lots/lib/archivos.ts;
// the backend can't trust the client's check.
const MaxArchivoFileBytes = 10_000_000

// MaxArchivosPerEntity bounds how many active fotos/planos a single loteo or
// lote may carry, so an entity can't accumulate uploads without limit.
const MaxArchivosPerEntity = 20

// archivoMimeTypes are the only content types StoreLoteoArchivo/StoreLoteArchivo
// accept. Kept narrow (photos plus PDF for scanned planos) since anything
// stored here is later served back to a browser.
var archivoMimeTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/webp":      true,
	"application/pdf": true,
}

func ValidArchivoCategoria(categoria string) bool {
	return categoria == ArchivoCategoriaFoto || categoria == ArchivoCategoriaPlano
}

func ValidArchivoMimeType(mimeType string) bool {
	return archivoMimeTypes[mimeType]
}

var (
	ErrInvalidArchivo  = &Error{Kind: KindInvalid, Code: "invalid_archivo", Message: "El archivo es inválido, de un tipo no admitido o supera el tamaño permitido"}
	ErrTooManyArchivos = &Error{Kind: KindInvalid, Code: "too_many_archivos", Message: "Se alcanzó el máximo de archivos permitidos"}
	ErrArchivoNotFound = &Error{Kind: KindNotFound, Code: "archivo_not_found", Message: "El archivo solicitado no existe"}
)

// NewArchivo is a foto or plano about to be recorded, once its bytes are
// stored in object storage under StorageKey.
type NewArchivo struct {
	Categoria    string
	StorageKey   string
	OriginalName string
	MimeType     string
	Sha256       string
}

// Archivo is a recorded foto or plano attached to a loteo or one of its
// lotes.
type Archivo struct {
	ID           string
	Categoria    string
	StorageKey   string
	OriginalName string
	MimeType     string
	Sha256       string
	CreatedAt    time.Time
}
