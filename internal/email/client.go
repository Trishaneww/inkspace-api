package email

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	secret  string
	log     *slog.Logger
}

func NewClient(baseURL, secret string, log *slog.Logger) *Client {
	return &Client{baseURL: baseURL, secret: secret, log: log}
}

func (c *Client) Post(path string, payload any) {
	if c.secret == "" {
		return
	}
	body, err := json.Marshal(payload)
	if err != nil {
		c.log.Warn("internal_email_marshal_failed", "path", path, "error", err)
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
		if err != nil {
			c.log.Warn("internal_email_request_failed", "path", path, "error", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-internal-secret", c.secret)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			c.log.Warn("internal_email_send_failed", "path", path, "error", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= http.StatusMultipleChoices {
			c.log.Warn("internal_email_rejected", "path", path, "status", resp.StatusCode)
		}
	}()
}
