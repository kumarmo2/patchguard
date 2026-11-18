package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testWebhookSecret = "test-webhook-secret"

func signWebhook(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

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
			request.Header.Set(githubSignatureHeader, signWebhook(testWebhookSecret, tt.payload))
			response := httptest.NewRecorder()
			handler(testWebhookSecret).ServeHTTP(response, request)
			if response.Code != tt.status {
				t.Errorf("status = %d, want %d", response.Code, tt.status)
			}
			if tt.body != "" && response.Body.String() != tt.body {
				t.Errorf("body = %q, want %q", response.Body.String(), tt.body)
			}
		})
	}
}

func TestWebhookAuthentication(t *testing.T) {
	// Published test vector from GitHub's "Validating webhook deliveries" docs.
	const secret = "It's a Secret to Everybody"
	const payload = "Hello, World!"
	const signature = "sha256=757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17"

	tests := []struct {
		name       string
		secret     string
		payload    string
		event      string
		signatures []string
		status     int
	}{
		{"valid GitHub example", secret, payload, "ping", []string{signature}, http.StatusOK},
		{"missing signature on ping", secret, payload, "ping", nil, http.StatusForbidden},
		{"missing signature before JSON decoding", secret, `{"number":"bad"}`, "pull_request", nil, http.StatusForbidden},
		{"wrong prefix", secret, payload, "ping", []string{"sha1=" + signature[len("sha256="):]}, http.StatusForbidden},
		{"short digest", secret, payload, "ping", []string{"sha256=aa"}, http.StatusForbidden},
		{"non-hex digest", secret, payload, "ping", []string{"sha256=" + strings.Repeat("z", 64)}, http.StatusForbidden},
		{"duplicate headers", secret, payload, "ping", []string{signature, signature}, http.StatusForbidden},
		{"tampered body", secret, payload + "\n", "ping", []string{signature}, http.StatusForbidden},
		{"wrong secret", testWebhookSecret, payload, "ping", []string{signature}, http.StatusForbidden},
		{"empty secret", "", payload, "ping", []string{signWebhook("", payload)}, http.StatusForbidden},
		{"oversized body", secret, strings.Repeat("x", maxWebhookBodyBytes+1), "ping", []string{signature}, http.StatusRequestEntityTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader(tt.payload))
			request.Header.Set("X-GitHub-Event", tt.event)
			for _, signature := range tt.signatures {
				request.Header.Add(githubSignatureHeader, signature)
			}
			response := httptest.NewRecorder()
			handler(tt.secret).ServeHTTP(response, request)
			if response.Code != tt.status {
				t.Errorf("status = %d, want %d", response.Code, tt.status)
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
	request.Header.Set(githubSignatureHeader, signWebhook(testWebhookSecret, payload))
	response := httptest.NewRecorder()
	handler(testWebhookSecret).ServeHTTP(response, request)

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
