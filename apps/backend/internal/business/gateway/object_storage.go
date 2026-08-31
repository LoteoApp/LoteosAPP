package gateway

import (
	"context"
	"io"
)

// StoredObject is an object read back from object storage. The caller owns
// Body and must close it.
type StoredObject struct {
	Body        io.ReadCloser
	ContentType string
	Size        int64
}

// ObjectStorage abstracts file storage so the business layer stores and
// reads files without depending on a concrete provider.
//
// A key is a slash-separated path relative to the bucket root, without a
	// leading slash (e.g. "loteos/42/dxf/version.dxf"). Implementations reject a
// key that is empty, absolute, longer than the provider allows, or that
// contains a "." or ".." segment, with domain.ErrInvalidObjectKey.
type ObjectStorage interface {
	// Put stores body under key, replacing any object already there. size
	// must be the exact byte length of body.
	//
	// body has to be seekable: signing the request means hashing the payload
	// and rewinding, and a retry replays it from the start. A caller holding
	// a plain stream (an HTTP request body, a pipe) has to materialize it
	// first — to a file or to memory — and that choice, with its memory cost,
	// belongs to the caller and not to this contract.
	Put(ctx context.Context, key string, body io.ReadSeeker, size int64, contentType string) error

	// Get returns the object stored under key. It returns
	// domain.ErrObjectNotFound when no object exists under that key.
	Get(ctx context.Context, key string) (*StoredObject, error)

	// Delete removes the object stored under key. Deleting a key that
	// holds no object is not an error.
	Delete(ctx context.Context, key string) error
}
