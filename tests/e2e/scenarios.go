package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/mattsolo1/grove-tend/pkg/assert"
	"github.com/mattsolo1/grove-tend/pkg/command"
	"github.com/mattsolo1/grove-tend/pkg/harness"
)

// NotifySystemScenario tests the 'notify system' command.
func NotifySystemScenario() *harness.Scenario {
	return &harness.Scenario{
		Name: "notify-system-command",
		Steps: []harness.Step{
			harness.NewStep("Run 'notify system'", func(ctx *harness.Context) error {
				notifyBinary := os.Getenv("NOTIFY_BINARY")
				if notifyBinary == "" {
					return fmt.Errorf("NOTIFY_BINARY environment variable not set")
				}
				
				cmd := command.New(notifyBinary, "system", "--title", "E2E Test", "Hello from tend!")
				result := cmd.Run()
				ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
				
				if err := assert.Equal(0, result.ExitCode, "notify system should exit successfully"); err != nil {
					return err
				}
				return assert.Contains(result.Stdout, "System notification sent:", "Output should confirm notification was sent")
			}),
		},
	}
}

// NotifyNtfyScenario tests the 'notify ntfy' command with a mock server.
func NotifyNtfyScenario() *harness.Scenario {
	var mockServer *httptest.Server
	var lastRequest struct {
		Body []byte
		Path string
		Headers http.Header
	}

	return &harness.Scenario{
		Name: "notify-ntfy-command",
		Steps: []harness.Step{
			harness.NewStep("Start mock ntfy server", func(ctx *harness.Context) error {
				// Start a mock ntfy server
				mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					lastRequest.Path = r.URL.Path
					lastRequest.Headers = r.Header
					lastRequest.Body, _ = io.ReadAll(r.Body)
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"success":true}`))
				}))
				ctx.Set("mock_server_url", mockServer.URL)
				ctx.Set("lastRequest", &lastRequest)
				return nil
			}),
			harness.NewStep("Run 'notify ntfy' with all options", func(ctx *harness.Context) error {
				notifyBinary := os.Getenv("NOTIFY_BINARY")
				if notifyBinary == "" {
					return fmt.Errorf("NOTIFY_BINARY environment variable not set")
				}
				
				serverURL := ctx.GetString("mock_server_url")
				
				cmd := command.New(notifyBinary, "ntfy",
					"--url", serverURL,
					"--topic", "test-topic",
					"--title", "Ntfy E2E Test",
					"--priority", "high",
					"--tags", "e2e,test",
					"Ntfy message from tend",
				)
				result := cmd.Run()
				ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
				
				if err := assert.Equal(0, result.ExitCode, "notify ntfy should exit successfully"); err != nil {
					return err
				}
				return assert.Contains(result.Stdout, "Ntfy notification sent to topic", "Output should confirm notification was sent")
			}),
			harness.NewStep("Verify API call to mock server", func(ctx *harness.Context) error {
				if err := assert.Equal("/test-topic", lastRequest.Path, "Request path should match ntfy topic"); err != nil {
					return err
				}

				// Verify headers
				if err := assert.Equal("high", lastRequest.Headers.Get("Priority"), "Priority header should be set"); err != nil {
					return err
				}
				if err := assert.Equal("Ntfy E2E Test", lastRequest.Headers.Get("Title"), "Title header should be set"); err != nil {
					return err
				}
				if err := assert.Equal("e2e,test", lastRequest.Headers.Get("Tags"), "Tags header should be set"); err != nil {
					return err
				}

				// Verify body
				bodyStr := string(lastRequest.Body)
				return assert.Equal("Ntfy message from tend", bodyStr, "Request body should contain the message")
			}),
			harness.NewStep("Cleanup mock server", func(ctx *harness.Context) error {
				if mockServer != nil {
					mockServer.Close()
				}
				return nil
			}),
		},
	}
}