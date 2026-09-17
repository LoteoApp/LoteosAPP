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

	if err := client.send(ctx, invite.To, "Tu cuenta en LoteosAPP", html); err != nil {
		return fmt.Errorf("send invite email: %w", err)
	}

	return nil
}

// SendPasswordReset implements gateway.Mailer.
func (client *Client) SendPasswordReset(ctx context.Context, reset gateway.PasswordResetEmail) error {
	html, err := renderPasswordReset(reset)
	if err != nil {
		return fmt.Errorf("render password reset email: %w", err)
	}

	if err := client.send(ctx, reset.To, "Restablecé tu contraseña de LoteosAPP", html); err != nil {
		return fmt.Errorf("send password reset email: %w", err)
	}

	return nil
}

func (client *Client) send(ctx context.Context, to, subject, html string) error {
	body, err := json.Marshal(map[string]any{
		"from":    client.from,
		"to":      []string{to},
		"subject": subject,
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
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var apiError struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(response.Body).Decode(&apiError)
		return fmt.Errorf("unexpected status %d: %s", response.StatusCode, apiError.Message)
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
<p>Este link es de un solo uso y vence a la hora de haberse generado.</p>
`))

func renderUserInvite(invite gateway.UserInviteEmail) (string, error) {
	var buffer bytes.Buffer
	if err := userInviteTemplate.Execute(&buffer, invite); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

// passwordResetTemplate sends only the link, never a password: the account's
// password doesn't change until the recipient opens it and confirms one of
// their own choosing.
var passwordResetTemplate = template.Must(template.New("password_reset").Parse(`
<p>Hola {{.Nombre}} {{.Apellido}},</p>
<p>Recibimos un pedido para restablecer la contraseña de tu cuenta en LoteosAPP.</p>
<p>Si fuiste vos, hacé click en el siguiente link para elegir una contraseña nueva:</p>
<p><a href="{{.ResetURL}}">{{.ResetURL}}</a></p>
<p>El link vence en 1 hora. Si no pediste este cambio, podés ignorar este mail.</p>
`))

func renderPasswordReset(reset gateway.PasswordResetEmail) (string, error) {
	var buffer bytes.Buffer
	if err := passwordResetTemplate.Execute(&buffer, reset); err != nil {
		return "", err
	}
	return buffer.String(), nil
}
