package supabase

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"loteosapp/backend/internal/business/domain"
)

// AdminClient manages users in a Supabase Auth project through its Admin
// REST API, authenticating with the project's service_role key. It
// implements gateway.IdentityProvider.
type AdminClient struct {
	httpClient     *http.Client
	baseURL        string
	serviceRoleKey string
}

func NewAdminClient(baseURL, serviceRoleKey string) *AdminClient {
	return &AdminClient{
		httpClient:     &http.Client{Timeout: 10 * time.Second},
		baseURL:        strings.TrimSuffix(baseURL, "/"),
		serviceRoleKey: serviceRoleKey,
	}
}

// CreateUser implements gateway.IdentityProvider. It stores rol under
// app_metadata.role so Verifier can read it back from the access token.
func (client *AdminClient) CreateUser(ctx context.Context, email, rol string) (string, string, error) {
	temporaryPassword, err := generateTemporaryPassword()
	if err != nil {
		return "", "", fmt.Errorf("generate temporary password: %w", err)
	}

	body, err := json.Marshal(map[string]any{
		"email":         email,
		"password":      temporaryPassword,
		"email_confirm": true,
		"app_metadata":  map[string]string{"role": rol},
	})
	if err != nil {
		return "", "", fmt.Errorf("encode user payload: %w", err)
	}

	request, err := client.newRequest(ctx, http.MethodPost, client.adminURL("/users"), body)
	if err != nil {
		return "", "", err
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return "", "", fmt.Errorf("create supabase user: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", "", createUserError(response)
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		return "", "", fmt.Errorf("decode created user: %w", err)
	}
	if created.ID == "" {
		return "", "", fmt.Errorf("create supabase user: missing id in response")
	}

	return created.ID, temporaryPassword, nil
}

// DeleteUser implements gateway.IdentityProvider.
func (client *AdminClient) DeleteUser(ctx context.Context, supabaseID string) error {
	request, err := client.newRequest(ctx, http.MethodDelete, client.adminURL("/users/"+supabaseID), nil)
	if err != nil {
		return err
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("delete supabase user: %w", err)
	}
	defer response.Body.Close()

	// GoTrue's docs promise 204, but the hosted API observed in practice
	// returns 200 with the deleted user's representation.
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete supabase user: %w", unexpectedStatus(response))
	}

	return nil
}

func (client *AdminClient) newRequest(ctx context.Context, method, url string, body []byte) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	request, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	// Kong (Supabase's API gateway) routes on apikey; GoTrue authorizes the
	// admin call from the same service_role JWT via Authorization.
	request.Header.Set("apikey", client.serviceRoleKey)
	request.Header.Set("Authorization", "Bearer "+client.serviceRoleKey)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	return request, nil
}

func (client *AdminClient) adminURL(path string) string {
	return client.baseURL + "/auth/v1/admin" + path
}

func unexpectedStatus(response *http.Response) error {
	return fmt.Errorf("unexpected status %d", response.StatusCode)
}

// createUserError maps a non-200 response from POST /admin/users to an
// error. GoTrue identifies a duplicate email through the error_code field
// (observed as "email_exists" with HTTP 422), not through the status code
// alone: other failures, like a weak password, also return 422.
func createUserError(response *http.Response) error {
	var apiError struct {
		ErrorCode string `json:"error_code"`
		Message   string `json:"msg"`
	}
	_ = json.NewDecoder(response.Body).Decode(&apiError)

	if apiError.ErrorCode == "email_exists" || apiError.ErrorCode == "user_already_exists" {
		return domain.ErrEmailEnUso
	}

	return fmt.Errorf("create supabase user: unexpected status %d, error_code %q: %s",
		response.StatusCode, apiError.ErrorCode, apiError.Message)
}

func generateTemporaryPassword() (string, error) {
	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
