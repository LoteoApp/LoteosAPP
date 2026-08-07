package keycloak

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"loteosapp/backend/internal/business/domain"
)

// AdminClient manages users in a Keycloak realm through its Admin REST API,
// authenticating as a confidential client's service account
// (client_credentials grant). It implements gateway.IdentityProvider.
type AdminClient struct {
	httpClient   *http.Client
	baseURL      string
	realm        string
	clientID     string
	clientSecret string

	mu          sync.Mutex
	cachedToken string
	tokenExpiry time.Time
}

func NewAdminClient(baseURL, realm, clientID, clientSecret string) *AdminClient {
	return &AdminClient{
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		realm:        realm,
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

// CreateUser implements gateway.IdentityProvider.
func (client *AdminClient) CreateUser(ctx context.Context, email, rol string) (string, string, error) {
	token, err := client.token(ctx)
	if err != nil {
		return "", "", fmt.Errorf("keycloak admin token: %w", err)
	}

	keycloakID, err := client.createUserAccount(ctx, token, email)
	if err != nil {
		return "", "", err
	}

	temporaryPassword, err := generateTemporaryPassword()
	if err != nil {
		client.deleteUser(ctx, token, keycloakID)
		return "", "", fmt.Errorf("generate temporary password: %w", err)
	}

	if err := client.setTemporaryPassword(ctx, token, keycloakID, temporaryPassword); err != nil {
		client.deleteUser(ctx, token, keycloakID)
		return "", "", err
	}

	if err := client.assignRealmRole(ctx, token, keycloakID, rol); err != nil {
		client.deleteUser(ctx, token, keycloakID)
		return "", "", err
	}

	return keycloakID, temporaryPassword, nil
}

// DeleteUser implements gateway.IdentityProvider.
func (client *AdminClient) DeleteUser(ctx context.Context, keycloakID string) error {
	token, err := client.token(ctx)
	if err != nil {
		return fmt.Errorf("keycloak admin token: %w", err)
	}

	return client.deleteUser(ctx, token, keycloakID)
}

func (client *AdminClient) createUserAccount(ctx context.Context, token, email string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"username":        email,
		"email":           email,
		"enabled":         true,
		"emailVerified":   true,
		"requiredActions": []string{"UPDATE_PASSWORD"},
	})
	if err != nil {
		return "", fmt.Errorf("encode user payload: %w", err)
	}

	request, err := client.newRequest(ctx, http.MethodPost, client.adminURL("/users"), token, body)
	if err != nil {
		return "", err
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("create keycloak user: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusConflict {
		return "", domain.ErrEmailEnUso
	}
	if response.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create keycloak user: %w", unexpectedStatus(response))
	}

	location := response.Header.Get("Location")
	keycloakID := location[strings.LastIndex(location, "/")+1:]
	if keycloakID == "" {
		return "", fmt.Errorf("create keycloak user: missing Location header")
	}

	return keycloakID, nil
}

func (client *AdminClient) setTemporaryPassword(ctx context.Context, token, keycloakID, password string) error {
	body, err := json.Marshal(map[string]any{
		"type":      "password",
		"value":     password,
		"temporary": true,
	})
	if err != nil {
		return fmt.Errorf("encode credential payload: %w", err)
	}

	request, err := client.newRequest(ctx, http.MethodPut, client.adminURL("/users/"+keycloakID+"/reset-password"), token, body)
	if err != nil {
		return err
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("set temporary password: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("set temporary password: %w", unexpectedStatus(response))
	}

	return nil
}

func (client *AdminClient) assignRealmRole(ctx context.Context, token, keycloakID, rol string) error {
	role, err := client.findRealmRole(ctx, token, rol)
	if err != nil {
		return err
	}

	body, err := json.Marshal([]map[string]any{role})
	if err != nil {
		return fmt.Errorf("encode role mapping payload: %w", err)
	}

	request, err := client.newRequest(ctx, http.MethodPost, client.adminURL("/users/"+keycloakID+"/role-mappings/realm"), token, body)
	if err != nil {
		return err
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("assign realm role: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("assign realm role: %w", unexpectedStatus(response))
	}

	return nil
}

func (client *AdminClient) findRealmRole(ctx context.Context, token, rol string) (map[string]any, error) {
	request, err := client.newRequest(ctx, http.MethodGet, client.adminURL("/roles/"+url.PathEscape(rol)), token, nil)
	if err != nil {
		return nil, err
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("look up realm role %q: %w", rol, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("look up realm role %q: %w", rol, unexpectedStatus(response))
	}

	var role struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(response.Body).Decode(&role); err != nil {
		return nil, fmt.Errorf("decode realm role %q: %w", rol, err)
	}

	return map[string]any{"id": role.ID, "name": role.Name}, nil
}

func (client *AdminClient) deleteUser(ctx context.Context, token, keycloakID string) error {
	request, err := client.newRequest(ctx, http.MethodDelete, client.adminURL("/users/"+keycloakID), token, nil)
	if err != nil {
		return err
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("delete keycloak user: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete keycloak user: %w", unexpectedStatus(response))
	}

	return nil
}

// token returns a cached client_credentials access token, fetching a new
// one when there is none or it is about to expire.
func (client *AdminClient) token(ctx context.Context) (string, error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.cachedToken != "" && time.Now().Before(client.tokenExpiry) {
		return client.cachedToken, nil
	}

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {client.clientID},
		"client_secret": {client.clientSecret},
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost,
		client.baseURL+"/realms/"+client.realm+"/protocol/openid-connect/token",
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := client.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", unexpectedStatus(response)
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}

	client.cachedToken = payload.AccessToken
	// Refresh a bit before actual expiry to avoid racing a token that
	// expires mid-request.
	client.tokenExpiry = time.Now().Add(time.Duration(payload.ExpiresIn)*time.Second - 10*time.Second)

	return client.cachedToken, nil
}

func (client *AdminClient) newRequest(ctx context.Context, method, url, token string, body []byte) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	request, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	return request, nil
}

func (client *AdminClient) adminURL(path string) string {
	return client.baseURL + "/admin/realms/" + client.realm + path
}

func unexpectedStatus(response *http.Response) error {
	return fmt.Errorf("unexpected status %d", response.StatusCode)
}

func generateTemporaryPassword() (string, error) {
	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
