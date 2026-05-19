package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Message struct {
	Content string `json:"content"`
}

func SendWebhook(ctx context.Context, webhookURL, message string) error {
	payload := Message{message}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encoding json: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		bodyBytes, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("bad response status code: %d", res.StatusCode)
		}
		return fmt.Errorf("bad response: %s", string(bodyBytes))
	}

	return nil
}
