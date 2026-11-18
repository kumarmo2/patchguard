package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetSecretRejectsWhitespaceOnlyPassword(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir) // Never run a real bw binary in this test.
	path := filepath.Join(dir, "password")
	if err := os.WriteFile(path, []byte(" \t\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	manager, err := NewSecretsManager(path)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := manager.GetSecret(t.Context(), "test-key"); err == nil || err.Error() != "password cannot be empty" {
		t.Fatalf("GetSecret error = %v, want password cannot be empty", err)
	}
}
