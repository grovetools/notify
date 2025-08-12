package notifiers

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type NtfyNotifier struct {
	URL    string
	Topic  string
	client *http.Client
}

func NewNtfyNotifier(url, topic string) *NtfyNotifier {
	return &NtfyNotifier{
		URL:   url,
		Topic: topic,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (n *NtfyNotifier) Send(title, message, priority string, tags []string) error {
	if n.URL == "" || n.Topic == "" {
		return fmt.Errorf("ntfy URL or topic not configured")
	}

	// ntfy.sh expects the topic in the URL path
	url := fmt.Sprintf("%s/%s", strings.TrimSuffix(n.URL, "/"), n.Topic)

	// Create request with message as body
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(message))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// ntfy.sh uses headers for metadata
	if title != "" {
		req.Header.Set("Title", title)
	}
	if priority != "" {
		req.Header.Set("Priority", priority)
	}
	if len(tags) > 0 {
		req.Header.Set("Tags", strings.Join(tags, ","))
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read response body for better error information
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		return fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(body[:n]))
	}

	return nil
}