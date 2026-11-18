package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebhook(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		status int
		body   string
	}{
		{"post webhook", http.MethodPost, "/webhooks/github", http.StatusOK, "OK\n"},
		{"wrong method", http.MethodGet, "/webhooks/github", http.StatusMethodNotAllowed, ""},
		{"wrong path", http.MethodPost, "/other", http.StatusNotFound, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.path, strings.NewReader(`{"test":true}`))
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
