package r2_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
	"loteosapp/backend/internal/infrastructure/storage/r2"
)

const testBucket = "loteos-files-test"

var _ gateway.ObjectStorage = (*r2.Client)(nil)

// fakeS3 serves the subset of the S3 API the client uses, addressed
// path-style as /<bucket>/<key>.
type fakeS3 struct {
	mu      sync.Mutex
	objects map[string][]byte
	types   map[string]string

	status    int
	errorCode string
	signed    bool
	deletes   int
}

func newFakeS3(t *testing.T) (*fakeS3, *r2.Client) {
	t.Helper()

	fake := &fakeS3{
		objects: make(map[string][]byte),
		types:   make(map[string]string),
	}

	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	client, err := r2.NewClient(r2.Config{
		Endpoint:        server.URL,
		Bucket:          testBucket,
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret-key",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	return fake, client
}

func (fake *fakeS3) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fake.mu.Lock()
	defer fake.mu.Unlock()

	fake.signed = strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ")

	if fake.status != 0 {
		writeS3Error(w, fake.status, fake.errorCode)
		return
	}

	key, ok := strings.CutPrefix(r.URL.EscapedPath(), "/"+testBucket+"/")
	if !ok {
		writeS3Error(w, http.StatusNotFound, "NoSuchBucket")
		return
	}

	switch r.Method {
	case http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeS3Error(w, http.StatusBadRequest, "IncompleteBody")
			return
		}
		fake.objects[key] = body
		fake.types[key] = r.Header.Get("Content-Type")
		w.Header().Set("ETag", `"fake-etag"`)
		w.WriteHeader(http.StatusOK)

	case http.MethodGet:
		body, found := fake.objects[key]
		if !found {
			writeS3Error(w, http.StatusNotFound, "NoSuchKey")
			return
		}
		w.Header().Set("Content-Type", fake.types[key])
		w.Header().Set("Content-Length", fmt.Sprint(len(body)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)

	case http.MethodDelete:
		fake.deletes++
		delete(fake.objects, key)
		w.WriteHeader(http.StatusNoContent)

	default:
		writeS3Error(w, http.StatusMethodNotAllowed, "MethodNotAllowed")
	}
}

func writeS3Error(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>%s</Code><Message>%s</Message></Error>`, code, code)
}

func (fake *fakeS3) stored(key string) ([]byte, string, bool) {
	fake.mu.Lock()
	defer fake.mu.Unlock()

	body, found := fake.objects[key]

	return body, fake.types[key], found
}

func (fake *fakeS3) failWith(status int) {
	fake.failWithCode(status, "InternalError")
}

func (fake *fakeS3) failWithCode(status int, code string) {
	fake.mu.Lock()
	defer fake.mu.Unlock()

	fake.status = status
	fake.errorCode = code
}

func TestPut(t *testing.T) {
	t.Parallel()

	t.Run("stores the body under the key with its content type", func(t *testing.T) {
		fake, client := newFakeS3(t)
		content := []byte("0\nSECTION\n2\nENTITIES\n0\nENDSEC\n")

		err := client.Put(context.Background(), "loteos/42/original.dxf", bytes.NewReader(content), int64(len(content)), "application/dxf")
		if err != nil {
			t.Fatalf("Put() error = %v", err)
		}

		body, contentType, found := fake.stored("loteos/42/original.dxf")
		if !found {
			t.Fatal("Put() stored nothing under the key")
		}
		if !bytes.Equal(body, content) {
			t.Errorf("stored body = %q, want %q", body, content)
		}
		if contentType != "application/dxf" {
			t.Errorf("stored content type = %q, want %q", contentType, "application/dxf")
		}
		if !fake.signed {
			t.Error("Put() sent an unsigned request, want a SigV4 Authorization header")
		}
	})

	t.Run("rejects a negative size", func(t *testing.T) {
		fake, client := newFakeS3(t)

		err := client.Put(context.Background(), "a.dxf", strings.NewReader("x"), -1, "")
		if !errors.Is(err, domain.ErrInvalidObjectSize) {
			t.Fatalf("Put() error = %v, want ErrInvalidObjectSize", err)
		}
		if _, _, found := fake.stored("a.dxf"); found {
			t.Error("Put() reached the bucket despite the invalid size")
		}
	})

	t.Run("reports an upstream failure as unavailable without leaking it", func(t *testing.T) {
		fake, client := newFakeS3(t)
		fake.failWith(http.StatusInternalServerError)

		err := client.Put(context.Background(), "a.dxf", strings.NewReader("x"), 1, "")

		assertUnavailable(t, err)
	})

	t.Run("reports a rejected oversized upload as an invalid size", func(t *testing.T) {
		fake, client := newFakeS3(t)
		fake.failWithCode(http.StatusBadRequest, "EntityTooLarge")

		err := client.Put(context.Background(), "huge.dxf", strings.NewReader("x"), 1, "")

		if !errors.Is(err, domain.ErrInvalidObjectSize) {
			t.Fatalf("Put() error = %v, want ErrInvalidObjectSize", err)
		}

		var domainErr *domain.Error
		if !errors.As(err, &domainErr) {
			t.Fatalf("Put() error = %v, want a *domain.Error", err)
		}
		if domainErr.Kind != domain.KindInvalid {
			t.Errorf("Put() error kind = %q, want %q", domainErr.Kind, domain.KindInvalid)
		}
	})
}

func TestNewClientRejectsUnsafeEndpoints(t *testing.T) {
	t.Parallel()

	unsafe := []struct {
		name     string
		endpoint string
	}{
		{name: "plain http to a remote host", endpoint: "http://account.r2.cloudflarestorage.com"},
		{name: "no scheme", endpoint: "account.r2.cloudflarestorage.com"},
		{name: "unsupported scheme", endpoint: "ftp://account.r2.cloudflarestorage.com"},
		{name: "credentials in the url", endpoint: "https://user:pass@account.r2.cloudflarestorage.com"},
		{name: "query string", endpoint: "https://account.r2.cloudflarestorage.com?x=1"},
		{name: "no host", endpoint: "https://"},
	}

	for _, tt := range unsafe {
		t.Run(tt.name, func(t *testing.T) {
			_, err := r2.NewClient(r2.Config{
				Endpoint:        tt.endpoint,
				Bucket:          testBucket,
				AccessKeyID:     "access-key",
				SecretAccessKey: "secret-key",
			})
			if err == nil {
				t.Fatalf("NewClient(%q) error = nil, want an error", tt.endpoint)
			}
		})
	}

	safe := []struct {
		name     string
		endpoint string
	}{
		{name: "https", endpoint: "https://account.r2.cloudflarestorage.com"},
		{name: "http on loopback for tests", endpoint: "http://127.0.0.1:9000"},
		{name: "http on localhost for tests", endpoint: "http://localhost:9000"},
	}

	for _, tt := range safe {
		t.Run(tt.name, func(t *testing.T) {
			_, err := r2.NewClient(r2.Config{
				Endpoint:        tt.endpoint,
				Bucket:          testBucket,
				AccessKeyID:     "access-key",
				SecretAccessKey: "secret-key",
			})
			if err != nil {
				t.Fatalf("NewClient(%q) error = %v", tt.endpoint, err)
			}
		})
	}
}

func TestGet(t *testing.T) {
	t.Parallel()

	t.Run("returns the stored object", func(t *testing.T) {
		_, client := newFakeS3(t)
		content := []byte("plano")
		if err := client.Put(context.Background(), "manzanas/7/plano.pdf", bytes.NewReader(content), int64(len(content)), "application/pdf"); err != nil {
			t.Fatalf("Put() error = %v", err)
		}

		object, err := client.Get(context.Background(), "manzanas/7/plano.pdf")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		defer object.Body.Close()

		body, err := io.ReadAll(object.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if !bytes.Equal(body, content) {
			t.Errorf("Get() body = %q, want %q", body, content)
		}
		if object.ContentType != "application/pdf" {
			t.Errorf("Get() content type = %q, want %q", object.ContentType, "application/pdf")
		}
		if object.Size != int64(len(content)) {
			t.Errorf("Get() size = %d, want %d", object.Size, len(content))
		}
	})

	t.Run("reports a missing key as not found", func(t *testing.T) {
		_, client := newFakeS3(t)

		_, err := client.Get(context.Background(), "missing.dxf")

		if !errors.Is(err, domain.ErrObjectNotFound) {
			t.Fatalf("Get() error = %v, want ErrObjectNotFound", err)
		}

		var domainErr *domain.Error
		if !errors.As(err, &domainErr) {
			t.Fatalf("Get() error = %v, want a *domain.Error", err)
		}
		if domainErr.Kind != domain.KindNotFound {
			t.Errorf("Get() error kind = %q, want %q", domainErr.Kind, domain.KindNotFound)
		}
	})

	t.Run("reports an upstream failure as unavailable without leaking it", func(t *testing.T) {
		fake, client := newFakeS3(t)
		fake.failWith(http.StatusInternalServerError)

		_, err := client.Get(context.Background(), "a.dxf")

		assertUnavailable(t, err)
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()

	t.Run("removes the stored object", func(t *testing.T) {
		fake, client := newFakeS3(t)
		if err := client.Put(context.Background(), "a.dxf", strings.NewReader("x"), 1, ""); err != nil {
			t.Fatalf("Put() error = %v", err)
		}

		if err := client.Delete(context.Background(), "a.dxf"); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		if _, _, found := fake.stored("a.dxf"); found {
			t.Error("Delete() left the object in the bucket")
		}
	})

	t.Run("succeeds on a key that holds no object", func(t *testing.T) {
		_, client := newFakeS3(t)

		if err := client.Delete(context.Background(), "never-existed.dxf"); err != nil {
			t.Fatalf("Delete() error = %v, want nil", err)
		}
	})

	t.Run("reports an upstream failure as unavailable without leaking it", func(t *testing.T) {
		fake, client := newFakeS3(t)
		fake.failWith(http.StatusInternalServerError)

		err := client.Delete(context.Background(), "a.dxf")

		assertUnavailable(t, err)
	})
}

func TestRejectsInvalidKeys(t *testing.T) {
	t.Parallel()

	invalid := []struct {
		name string
		key  string
	}{
		{name: "empty", key: ""},
		{name: "absolute", key: "/loteos/a.dxf"},
		{name: "parent traversal", key: "loteos/../../etc/passwd"},
		{name: "parent traversal at the end", key: "loteos/.."},
		{name: "current directory segment", key: "loteos/./a.dxf"},
		{name: "empty segment", key: "loteos//a.dxf"},
		{name: "trailing slash", key: "loteos/"},
		{name: "control character", key: "loteos/a\x00.dxf"},
		{name: "newline", key: "loteos/a\n.dxf"},
		{name: "longer than the S3 limit", key: strings.Repeat("k", 1025)},
	}

	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			fake, client := newFakeS3(t)
			ctx := context.Background()

			if err := client.Put(ctx, tt.key, strings.NewReader("x"), 1, ""); !errors.Is(err, domain.ErrInvalidObjectKey) {
				t.Errorf("Put() error = %v, want ErrInvalidObjectKey", err)
			}
			if _, err := client.Get(ctx, tt.key); !errors.Is(err, domain.ErrInvalidObjectKey) {
				t.Errorf("Get() error = %v, want ErrInvalidObjectKey", err)
			}
			if err := client.Delete(ctx, tt.key); !errors.Is(err, domain.ErrInvalidObjectKey) {
				t.Errorf("Delete() error = %v, want ErrInvalidObjectKey", err)
			}

			fake.mu.Lock()
			defer fake.mu.Unlock()
			if len(fake.objects) > 0 || fake.deletes > 0 {
				t.Error("an invalid key reached the bucket, want it rejected before the request")
			}
		})
	}
}

func TestNewClientRequiresEverySetting(t *testing.T) {
	t.Parallel()

	complete := r2.Config{
		Endpoint:        "https://account.r2.cloudflarestorage.com",
		Bucket:          testBucket,
		AccessKeyID:     "access-key",
		SecretAccessKey: "super-secret-key",
	}

	incomplete := []struct {
		name    string
		mutate  func(*r2.Config)
		wantHas string
	}{
		{name: "missing endpoint", mutate: func(c *r2.Config) { c.Endpoint = "" }, wantHas: "endpoint"},
		{name: "missing bucket", mutate: func(c *r2.Config) { c.Bucket = "" }, wantHas: "bucket"},
		{name: "missing access key id", mutate: func(c *r2.Config) { c.AccessKeyID = "" }, wantHas: "access key id"},
		{name: "missing secret access key", mutate: func(c *r2.Config) { c.SecretAccessKey = "" }, wantHas: "secret access key"},
	}

	for _, tt := range incomplete {
		t.Run(tt.name, func(t *testing.T) {
			cfg := complete
			tt.mutate(&cfg)

			_, err := r2.NewClient(cfg)
			if err == nil {
				t.Fatal("NewClient() error = nil, want an error")
			}
			if !strings.Contains(err.Error(), tt.wantHas) {
				t.Errorf("NewClient() error = %q, want it to mention %q", err, tt.wantHas)
			}
			if strings.Contains(err.Error(), complete.SecretAccessKey) {
				t.Errorf("NewClient() error leaks the secret access key: %q", err)
			}
		})
	}

	t.Run("accepts a complete config", func(t *testing.T) {
		if _, err := r2.NewClient(complete); err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
	})
}

func assertUnavailable(t *testing.T, err error) {
	t.Helper()

	if !errors.Is(err, domain.ErrStorageUnavailable) {
		t.Fatalf("error = %v, want ErrStorageUnavailable", err)
	}

	var domainErr *domain.Error
	if !errors.As(err, &domainErr) {
		t.Fatalf("error = %v, want a *domain.Error", err)
	}
	if domainErr.Kind != domain.KindUnavailable {
		t.Errorf("error kind = %q, want %q", domainErr.Kind, domain.KindUnavailable)
	}
	if domainErr.Cause == nil {
		t.Error("error carries no Cause, want the upstream failure attached for logging")
	}
	if domainErr.Message != domain.ErrStorageUnavailable.Message {
		t.Errorf("error message = %q, want the generic %q", domainErr.Message, domain.ErrStorageUnavailable.Message)
	}
}
