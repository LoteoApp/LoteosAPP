package r2

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

const (
	// R2 has no regions, but the S3 signing algorithm still needs one.
	region = "auto"

	maxAttempts = 3
	maxKeyBytes = 1024

	// The SDK default of 20s outlasts the timeout of the request the caller
	// is serving, turning a retry into a hang the client never sees resolved.
	maxBackoff = 2 * time.Second
)

// Config holds everything needed to reach one R2 bucket. Endpoint is the
// account-level S3 endpoint (https://<account-id>.r2.cloudflarestorage.com).
type Config struct {
	Endpoint        string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
}

// Client stores files in a Cloudflare R2 bucket over its S3-compatible API.
// It implements gateway.ObjectStorage.
type Client struct {
	s3     *s3.Client
	bucket string
}

func NewClient(cfg Config) (*Client, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	awsConfig := aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		HTTPClient:  newHTTPClient(),
		Retryer: func() aws.Retryer {
			return retry.NewStandard(func(options *retry.StandardOptions) {
				options.MaxAttempts = maxAttempts
				options.MaxBackoff = maxBackoff
			})
		},
	}

	client := s3.NewFromConfig(awsConfig, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(cfg.Endpoint)
		options.UsePathStyle = true
	})

	return &Client{s3: client, bucket: cfg.Bucket}, nil
}

func (client *Client) Put(ctx context.Context, key string, body io.ReadSeeker, size int64, contentType string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if size < 0 {
		return domain.ErrInvalidObjectSize
	}

	input := &s3.PutObjectInput{
		Bucket:        aws.String(client.bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentLength: aws.Int64(size),
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}

	if _, err := client.s3.PutObject(ctx, input); err != nil {
		if hasErrorCode(err, "EntityTooLarge", "MaxMessageLengthExceeded") {
			return domain.ErrInvalidObjectSize.WithCause(err)
		}

		return unavailable("put object", key, err)
	}

	return nil
}

func (client *Client) Get(ctx context.Context, key string) (*gateway.StoredObject, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	output, err := client.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(client.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, domain.ErrObjectNotFound.WithCause(err)
		}

		return nil, unavailable("get object", key, err)
	}

	return &gateway.StoredObject{
		Body:        output.Body,
		ContentType: aws.ToString(output.ContentType),
		Size:        aws.ToInt64(output.ContentLength),
	}, nil
}

func (client *Client) Delete(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	_, err := client.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(client.bucket),
		Key:    aws.String(key),
	})
	if err != nil && !isNotFound(err) {
		return unavailable("delete object", key, err)
	}

	return nil
}

func (cfg Config) validate() error {
	missing := make([]string, 0, 4)
	if cfg.Endpoint == "" {
		missing = append(missing, "endpoint")
	}
	if cfg.Bucket == "" {
		missing = append(missing, "bucket")
	}
	if cfg.AccessKeyID == "" {
		missing = append(missing, "access key id")
	}
	if cfg.SecretAccessKey == "" {
		missing = append(missing, "secret access key")
	}

	if len(missing) > 0 {
		return fmt.Errorf("r2 config: missing %s", strings.Join(missing, ", "))
	}

	return validateEndpoint(cfg.Endpoint)
}

// validateEndpoint refuses an endpoint that would send objects and signed
// headers somewhere unintended. Plain HTTP is allowed only against loopback,
// which is what the tests point at; anything else must be HTTPS.
func validateEndpoint(endpoint string) error {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("r2 config: endpoint is not a valid URL: %w", err)
	}
	if parsed.Host == "" {
		return fmt.Errorf("r2 config: endpoint %q has no host", endpoint)
	}
	if parsed.User != nil {
		return fmt.Errorf("r2 config: endpoint must not carry credentials")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("r2 config: endpoint must not carry a query or fragment")
	}

	switch parsed.Scheme {
	case "https":
		return nil
	case "http":
		if isLoopback(parsed.Hostname()) {
			return nil
		}

		return fmt.Errorf("r2 config: endpoint must use https, got %q", endpoint)
	default:
		return fmt.Errorf("r2 config: endpoint scheme must be https, got %q", parsed.Scheme)
	}
}

func isLoopback(hostname string) bool {
	if hostname == "localhost" {
		return true
	}

	address := net.ParseIP(hostname)

	return address != nil && address.IsLoopback()
}

// newHTTPClient bounds connection setup and the wait for response headers,
// but not the whole exchange: a global timeout would abort a large upload
// that is still making progress. The caller's context bounds the total.
func newHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			ExpectContinueTimeout: time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConns:          100,
			ForceAttemptHTTP2:     true,
		},
	}
}

func validateKey(key string) error {
	if key == "" || len(key) > maxKeyBytes {
		return domain.ErrInvalidObjectKey
	}
	if strings.ContainsFunc(key, func(r rune) bool { return r < ' ' || r == 0x7f }) {
		return domain.ErrInvalidObjectKey
	}

	for _, segment := range strings.Split(key, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return domain.ErrInvalidObjectKey
		}
	}

	return nil
}

func isNotFound(err error) bool {
	var noSuchKey *types.NoSuchKey
	if errors.As(err, &noSuchKey) {
		return true
	}

	// HeadObject-style 404s, and R2's own answer to a missing key, arrive as
	// a bare "NotFound" that never decodes into types.NoSuchKey.
	return hasErrorCode(err, "NoSuchKey", "NotFound")
}

func hasErrorCode(err error, codes ...string) bool {
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}

	return slices.Contains(codes, apiErr.ErrorCode())
}

func unavailable(operation, key string, cause error) error {
	return domain.ErrStorageUnavailable.WithCause(fmt.Errorf("r2 %s %q: %w", operation, key, cause))
}
