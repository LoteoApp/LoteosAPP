package supabase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"loteosapp/backend/internal/business/domain"
)

// AdminClient manages users in a Supabase Auth project through its Admin
// REST API, authenticating with the project's service_role key. It
// implements gateway.IdentityProvider.
type AdminClient struct {
	httpClient        *http.Client
	baseURL           string
	serviceRoleKey    string
	inviteRedirectURL string
}

// NewAdminClient builds an AdminClient. inviteRedirectURL is the frontend
// page that consumes an invite link (its token_hash and type fragment
// params); it's appended by this client, not by Supabase, so Supabase's own
// "Redirect URLs" allow list never needs to know about it.
func NewAdminClient(baseURL, serviceRoleKey, inviteRedirectURL string) *AdminClient {
	return &AdminClient{
		httpClient:        &http.Client{Timeout: 10 * time.Second},
		baseURL:           strings.TrimSuffix(baseURL, "/"),
		serviceRoleKey:    serviceRoleKey,
		inviteRedirectURL: inviteRedirectURL,
	}
}

// CreateUser implements gateway.IdentityProvider. It creates the account
// without a password (GoTrue leaves it unconfirmed until the invite link
// below is redeemed) and stores rol under app_metadata.role so Verifier can
// read it back from the access token. generate_link's request body has no
// app_metadata field, so that's a second call; if it fails, the just-created
// account is deleted so no passwordless, roleless account is left behind.
func (client *AdminClient) CreateUser(ctx context.Context, email, rol string) (string, string, error) {
	supabaseID, inviteURL, err := client.generateLink(ctx, "invite", email)
	if err != nil {
		return "", "", err
	}

	if err := client.putAppMetadata(ctx, supabaseID, rol); err != nil {
		if deleteErr := client.DeleteUser(ctx, supabaseID); deleteErr != nil {
			return "", "", fmt.Errorf("set app_metadata: %w (compensating delete also failed: %v)", err, deleteErr)
		}
		return "", "", fmt.Errorf("set app_metadata: %w", err)
	}

	return supabaseID, inviteURL, nil
}

// GenerateInviteLink implements gateway.IdentityProvider.
func (client *AdminClient) GenerateInviteLink(ctx context.Context, email string) (string, error) {
	_, inviteURL, err := client.generateLink(ctx, "invite", email)
	return inviteURL, err
}

const (
	usersPageSize = 1000
	maxUsersPages = 100
)

// ConfirmedAccountIDs implements gateway.IdentityProvider. It pages through
// GET /admin/users and keeps the accounts whose email_confirmed_at is set,
// which GoTrue fills in when an invite is redeemed with verifyOtp.
func (client *AdminClient) ConfirmedAccountIDs(ctx context.Context) (map[string]bool, error) {
	confirmed := map[string]bool{}

	for page := 1; page <= maxUsersPages; page++ {
		accounts, err := client.listUsersPage(ctx, page)
		if err != nil {
			return nil, err
		}
		for _, account := range accounts {
			if account.EmailConfirmedAt != nil {
				confirmed[account.ID] = true
			}
		}
		if len(accounts) < usersPageSize {
			return confirmed, nil
		}
	}

	return nil, fmt.Errorf("list supabase users: more than %d pages", maxUsersPages)
}

type supabaseAccount struct {
	ID               string  `json:"id"`
	EmailConfirmedAt *string `json:"email_confirmed_at"`
}

func (client *AdminClient) listUsersPage(ctx context.Context, page int) ([]supabaseAccount, error) {
	request, err := client.newRequest(ctx, http.MethodGet,
		client.adminURL(fmt.Sprintf("/users?page=%d&per_page=%d", page, usersPageSize)), nil)
	if err != nil {
		return nil, err
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("list supabase users: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list supabase users: %w", unexpectedStatus(response))
	}

	var listed struct {
		Users []supabaseAccount `json:"users"`
	}
	if err := json.NewDecoder(response.Body).Decode(&listed); err != nil {
		return nil, fmt.Errorf("decode supabase users: %w", err)
	}

	return listed.Users, nil
}

// generateLink calls POST /admin/generate_link and returns the created (or
// existing, for a resend) user's id plus an invite URL for
// client.inviteRedirectURL carrying that link's one-time token_hash. It
// deliberately doesn't pass redirect_to: the token is extracted here instead
// of relying on Supabase's own verify-and-redirect endpoint, so opening the
// mail's link never consumes the token by itself — only the frontend calling
// verifyOtp does, once the recipient actually submits a new password.
func (client *AdminClient) generateLink(ctx context.Context, linkType, email string) (string, string, error) {
	body, err := json.Marshal(map[string]any{
		"type":  linkType,
		"email": email,
	})
	if err != nil {
		return "", "", fmt.Errorf("encode generate_link payload: %w", err)
	}

	request, err := client.newRequest(ctx, http.MethodPost, client.adminURL("/generate_link"), body)
	if err != nil {
		return "", "", err
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return "", "", fmt.Errorf("generate link: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", "", createUserError(response)
	}

	var created struct {
		ID          string `json:"id"`
		HashedToken string `json:"hashed_token"`
	}
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		return "", "", fmt.Errorf("decode generate_link response: %w", err)
	}
	if created.ID == "" || created.HashedToken == "" {
		return "", "", fmt.Errorf("generate link: missing id or hashed_token in response")
	}

	inviteURL, err := client.buildInviteURL(linkType, created.HashedToken)
	if err != nil {
		return "", "", err
	}

	return created.ID, inviteURL, nil
}

// buildInviteURL puts token_hash and type in the URL fragment, not the
// query string: a fragment never leaves the browser, so it's absent from
// proxy/nginx access logs and from browser history synced elsewhere,
// unlike a query param.
func (client *AdminClient) buildInviteURL(linkType, tokenHash string) (string, error) {
	parsed, err := url.Parse(client.inviteRedirectURL)
	if err != nil {
		return "", fmt.Errorf("parse invite redirect URL: %w", err)
	}
	fragment := url.Values{}
	fragment.Set("token_hash", tokenHash)
	fragment.Set("type", linkType)
	parsed.Fragment = fragment.Encode()
	return parsed.String(), nil
}

// putAppMetadata sets app_metadata.role on an existing account.
// app_metadata (unlike user_metadata) can't be edited by the account's own
// owner, which is why the domain role lives there and not in the data field
// generate_link accepts.
func (client *AdminClient) putAppMetadata(ctx context.Context, supabaseID, rol string) error {
	body, err := json.Marshal(map[string]any{
		"app_metadata": map[string]string{"role": rol},
	})
	if err != nil {
		return fmt.Errorf("encode app_metadata payload: %w", err)
	}

	request, err := client.newRequest(ctx, http.MethodPut, client.adminURL("/users/"+supabaseID), body)
	if err != nil {
		return err
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return unexpectedStatus(response)
	}
	return nil
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

// createUserError maps a non-200 response from POST /admin/generate_link to
// an error. GoTrue identifies a duplicate, already-confirmed email through
// the error_code field (observed as "email_exists" with HTTP 422 on
// POST /admin/users; generate_link is assumed to share the same code) rather
// than through the status code alone.
func createUserError(response *http.Response) error {
	var apiError struct {
		ErrorCode string `json:"error_code"`
		Message   string `json:"msg"`
	}
	_ = json.NewDecoder(response.Body).Decode(&apiError)

	if apiError.ErrorCode == "email_exists" || apiError.ErrorCode == "user_already_exists" {
		return domain.ErrEmailEnUso
	}

	return fmt.Errorf("generate link: unexpected status %d, error_code %q: %s",
		response.StatusCode, apiError.ErrorCode, apiError.Message)
}
