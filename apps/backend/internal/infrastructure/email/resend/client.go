// Package resend sends transactional email through the Resend REST API
// (https://resend.com/docs/api-reference/emails/send-email). A plain
// net/http client is used instead of Resend's SDK: the API is a single
// simple JSON endpoint, so a dedicated dependency isn't worth adding.
package resend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"loteosapp/backend/internal/business/gateway"
)

const defaultBaseURL = "https://api.resend.com"

// Config is the sender identity and credential for the Resend account.
type Config struct {
	APIKey    string
	FromEmail string
	FromName  string
	// BaseURL overrides the Resend API root; leave empty in production, set
	// only in tests to point at an httptest.Server.
	BaseURL string
}

func (cfg Config) validate() error {
	var missing []string
	if cfg.APIKey == "" {
		missing = append(missing, "APIKey")
	}
	if cfg.FromEmail == "" {
		missing = append(missing, "FromEmail")
	}
	if len(missing) > 0 {
		return fmt.Errorf("resend: missing %s", strings.Join(missing, ", "))
	}
	return nil
}

// Client implements gateway.Mailer against the Resend API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	from       string
}

func NewClient(cfg Config) (*Client, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	from := cfg.FromEmail
	if cfg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", cfg.FromName, cfg.FromEmail)
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		apiKey:     cfg.APIKey,
		from:       from,
	}, nil
}

// SendUserInvite implements gateway.Mailer.
func (client *Client) SendUserInvite(ctx context.Context, invite gateway.UserInviteEmail) error {
	html, err := renderUserInvite(invite)
	if err != nil {
		return fmt.Errorf("render invite email: %w", err)
	}

	body, err := json.Marshal(map[string]any{
		"from":    client.from,
		"to":      []string{invite.To},
		"subject": "Tu cuenta en LoteosAPP",
		"html":    html,
	})
	if err != nil {
		return fmt.Errorf("encode email payload: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+"/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+client.apiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := client.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send invite email: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var apiError struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(response.Body).Decode(&apiError)
		return fmt.Errorf("send invite email: unexpected status %d: %s", response.StatusCode, apiError.Message)
	}

	return nil
}

// userInviteTemplate is intentionally plain: no styling assets to host, no
// external images to break in an email client. html/template auto-escapes
// every field, so Nombre/Apellido can't inject markup into the message.
var userInviteTemplate = template.Must(template.New("user_invite").Parse(`
<p>Hola {{.Nombre}} {{.Apellido}},</p>
<p>Se creó tu cuenta en LoteosAPP con el rol <strong>{{.Rol}}</strong> ({{.To}}).</p>
<p><a href="{{.InviteURL}}">Activá tu cuenta</a> para elegir tu contraseña y empezar a usarla.</p>
<p>Este link es de un solo uso y vence pasado un tiempo. Si ya no funciona, pedile a un administrador que te reenvíe uno nuevo.</p>
`))

func renderUserInvite(invite gateway.UserInviteEmail) (string, error) {
	var buffer bytes.Buffer
	if err := userInviteTemplate.Execute(&buffer, invite); err != nil {
		return "", err
	}
	return buffer.String(), nil
}
