package domain

var (
	ErrObjectNotFound     = &Error{Kind: KindNotFound, Code: "object_not_found", Message: "El archivo solicitado no existe"}
	ErrInvalidObjectKey   = &Error{Kind: KindInvalid, Code: "invalid_object_key", Message: "Ruta de archivo inválida"}
	ErrInvalidObjectSize  = &Error{Kind: KindInvalid, Code: "invalid_object_size", Message: "Tamaño de archivo inválido"}
	ErrStorageUnavailable = &Error{Kind: KindUnavailable, Code: "storage_unavailable", Message: "El almacenamiento de archivos no está disponible"}
)
