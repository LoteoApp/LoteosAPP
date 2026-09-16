package r2_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/infrastructure/storage/r2"
)

// TestClientIntegration is an integration test: it needs a real Cloudflare R2
// bucket (see docs/secrets.md#cloudflare-r2) and is skipped when the
// CLOUDFLARE_R2_* variables are not set. Every object it writes lives under
// the integration-test/ prefix and is deleted before the test ends.
func TestClientIntegration(t *testing.T) {
	config := r2.Config{
		Endpoint:        os.Getenv("CLOUDFLARE_R2_ENDPOINT"),
		Bucket:          os.Getenv("CLOUDFLARE_R2_BUCKET_NAME"),
		AccessKeyID:     os.Getenv("CLOUDFLARE_R2_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("CLOUDFLARE_R2_SECRET_ACCESS_KEY"),
	}
	if config.Endpoint == "" || config.Bucket == "" || config.AccessKeyID == "" || config.SecretAccessKey == "" {
		t.Skip("CLOUDFLARE_R2_* not set, skipping R2 integration test")
	}

	client, err := r2.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	key := "integration-test/" + randomHex(t) + ".dxf"
	content := []byte("0\nSECTION\n2\nENTITIES\n0\nENDSEC\n0\nEOF\n")

	t.Cleanup(func() {
		if err := client.Delete(context.Background(), key); err != nil {
			t.Errorf("cleanup Delete(%q) error = %v", key, err)
		}
	})

	if err := client.Put(ctx, key, bytes.NewReader(content), int64(len(content)), "application/dxf"); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	object, err := client.Get(ctx, key)
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
	if object.ContentType != "application/dxf" {
		t.Errorf("Get() content type = %q, want %q", object.ContentType, "application/dxf")
	}
	if object.Size != int64(len(content)) {
		t.Errorf("Get() size = %d, want %d", object.Size, len(content))
	}

	if err := client.Delete(ctx, key); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := client.Get(ctx, key); !errors.Is(err, domain.ErrObjectNotFound) {
		t.Fatalf("Get() after Delete() error = %v, want ErrObjectNotFound", err)
	}
}

func randomHex(t *testing.T) string {
	t.Helper()

	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		t.Fatalf("rand.Read() error = %v", err)
	}

	return hex.EncodeToString(raw)
}
