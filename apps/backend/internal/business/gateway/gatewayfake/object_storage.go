package gatewayfake

import (
	"bytes"
	"context"
	"io"
	"sync"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// ObjectStorage is an in-memory gateway.ObjectStorage for tests.
type ObjectStorage struct {
	PutErr    error
	GetErr    error
	DeleteErr error

	mu      sync.Mutex
	objects map[string]storedObject

	PutCalls    int
	GetCalls    int
	DeleteCalls int
}

type storedObject struct {
	body        []byte
	contentType string
}

func (fake *ObjectStorage) Put(_ context.Context, key string, body io.ReadSeeker, _ int64, contentType string) error {
	fake.mu.Lock()
	defer fake.mu.Unlock()

	fake.PutCalls++
	if fake.PutErr != nil {
		return fake.PutErr
	}

	content, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	if fake.objects == nil {
		fake.objects = make(map[string]storedObject)
	}
	fake.objects[key] = storedObject{body: content, contentType: contentType}

	return nil
}

func (fake *ObjectStorage) Get(_ context.Context, key string) (*gateway.StoredObject, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()

	fake.GetCalls++
	if fake.GetErr != nil {
		return nil, fake.GetErr
	}

	object, found := fake.objects[key]
	if !found {
		return nil, domain.ErrObjectNotFound
	}

	return &gateway.StoredObject{
		Body:        io.NopCloser(bytes.NewReader(object.body)),
		ContentType: object.contentType,
		Size:        int64(len(object.body)),
	}, nil
}

func (fake *ObjectStorage) Delete(_ context.Context, key string) error {
	fake.mu.Lock()
	defer fake.mu.Unlock()

	fake.DeleteCalls++
	if fake.DeleteErr != nil {
		return fake.DeleteErr
	}

	delete(fake.objects, key)

	return nil
}

// Contents returns the bytes stored under key, and whether it exists.
func (fake *ObjectStorage) Contents(key string) ([]byte, bool) {
	fake.mu.Lock()
	defer fake.mu.Unlock()

	object, found := fake.objects[key]

	return object.body, found
}
