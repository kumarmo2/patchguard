package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebhook(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		path    string
		event   string
		payload string
		status  int
		body    string
	}{
		{"post webhook", http.MethodPost, "/webhooks/github", "", `{"test":true}`, http.StatusOK, "OK\n"},
		{"ignore other events", http.MethodPost, "/webhooks/github", "ping", `not JSON`, http.StatusOK, "OK\n"},
		{"invalid PR payload", http.MethodPost, "/webhooks/github", "pull_request", `{"number":"bad"}`, http.StatusBadRequest, ""},
		{"null PR payload", http.MethodPost, "/webhooks/github", "pull_request", `null`, http.StatusBadRequest, ""},
		{"trailing JSON", http.MethodPost, "/webhooks/github", "pull_request", `{} {}`, http.StatusBadRequest, ""},
		{"wrong method", http.MethodGet, "/webhooks/github", "", "", http.StatusMethodNotAllowed, ""},
		{"wrong path", http.MethodPost, "/other", "", "", http.StatusNotFound, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.payload))
			request.Header.Set("X-GitHub-Event", tt.event)
			response := httptest.NewRecorder()
			handler().ServeHTTP(response, request)
			if response.Code != tt.status {
				t.Errorf("status = %d, want %d", response.Code, tt.status)
			}
			if tt.body != "" && response.Body.String() != tt.body {
				t.Errorf("body = %q, want %q", response.Body.String(), tt.body)
			}
		})
	}
}

func TestWebhookLogsPullRequestMetadata(t *testing.T) {
	var logs bytes.Buffer
	previousOutput := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previousOutput) })

	const payload = `{
		"action":"opened", "number":42,
		"repository":{"full_name":"owner/repo"},
		"pull_request":{"title":"private-title", "body":"private-description"}
	}`
	request := httptest.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader(payload))
	request.Header.Set("X-GitHub-Event", "pull_request")
	request.Header.Set("X-GitHub-Delivery", "test-delivery")
	response := httptest.NewRecorder()
	handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != "OK\n" {
		t.Fatalf("unexpected response: %d %q", response.Code, response.Body.String())
	}
	const metadata = `event=pull_request delivery="test-delivery" action="opened" repository="owner/repo" number=42`
	if !strings.Contains(logs.String(), metadata) {
		t.Errorf("log missing PR metadata: %q", logs.String())
	}
	if strings.Contains(logs.String(), "private-title") || strings.Contains(logs.String(), "private-description") {
		t.Errorf("log contains PR text: %q", logs.String())
	}
}
