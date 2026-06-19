package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/trishaneupnexx/inkspace-api/internal/config"
)

type Sender interface {
	Send(ctx context.Context, to, body string) error
}

func NewSender(cfg *config.Config, log *slog.Logger) Sender {
	if cfg.AppEnv == "production" && cfg.VonageAPIKey != "" && cfg.VonageAPISecret != "" {
		return &vonageSender{
			apiKey:    cfg.VonageAPIKey,
			apiSecret: cfg.VonageAPISecret,
			from:      cfg.VonageFromName,
		}
	}
	return &consoleSender{log: log}
}

type consoleSender struct {
	log *slog.Logger
}

func (c *consoleSender) Send(_ context.Context, to, body string) error {
	fmt.Fprintf(os.Stderr,
		"\n"+
			"📱 ═════════════════════════════════════════════════════ 📱\n"+
			"   📱📱📱  SMS (console only — not sent)  📱📱📱\n"+
			"      to:   %s\n"+
			"      body: %s\n"+
			"📱 ═════════════════════════════════════════════════════ 📱\n",
		to, body,
	)
	return nil
}

type vonageSender struct {
	apiKey    string
	apiSecret string
	from      string
}

func (v *vonageSender) Send(ctx context.Context, to, body string) error {
	form := url.Values{}
	form.Set("api_key", v.apiKey)
	form.Set("api_secret", v.apiSecret)
	form.Set("from", v.from)
	form.Set("to", digitsOnly(to))
	form.Set("text", body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://rest.nexmo.com/sms/json", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var out struct {
		Messages []struct {
			Status    string `json:"status"`
			ErrorText string `json:"error-text"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if len(out.Messages) == 0 {
		return fmt.Errorf("vonage send failed: empty response")
	}
	if out.Messages[0].Status != "0" {
		return fmt.Errorf("vonage send failed: %s", out.Messages[0].ErrorText)
	}
	return nil
}

func digitsOnly(phone string) string {
	var b strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
