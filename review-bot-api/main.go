package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	app_utils "github.com/kumarmo2/patchguard/review-bot-api/utils"
)

const CLOUDFLARE_SECRET_NAME string = "CLOUDFLARE_SECRET_NAME"
const SECRET_PASSWORD_FILE string = "SECRET_PASSWORD_FILE"

const (
	githubSignatureHeader    = "X-Hub-Signature-256"
	defaultWebhookSecretItem = "github-review-app-v1"
	maxWebhookBodyBytes      = 25_000_000
)

func handler(webhookSecret string) http.Handler {
	key := []byte(webhookSecret)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /webhooks/github", func(w http.ResponseWriter, r *http.Request) {
		signatures := r.Header.Values(githubSignatureHeader)
		if len(key) == 0 || len(signatures) != 1 {
			http.Error(w, "invalid webhook signature", http.StatusForbidden)
			return
		}
		digest, ok := strings.CutPrefix(signatures[0], "sha256=")
		if !ok || len(digest) != hex.EncodedLen(sha256.Size) {
			http.Error(w, "invalid webhook signature", http.StatusForbidden)
			return
		}
		signature, err := hex.DecodeString(digest)
		if err != nil {
			http.Error(w, "invalid webhook signature", http.StatusForbidden)
			return
		}

		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxWebhookBodyBytes))
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				http.Error(w, "webhook body too large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "failed to read webhook body", http.StatusBadRequest)
			return
		}

		// Verify the signature over the raw payload before decoding JSON.
		// https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries
		mac := hmac.New(sha256.New, key)
		mac.Write(body)
		if !hmac.Equal(signature, mac.Sum(nil)) {
			http.Error(w, "invalid webhook signature", http.StatusForbidden)
			return
		}

		if r.Header.Get("X-GitHub-Event") != "pull_request" {
			fmt.Fprint(w, "OK\n")
			return
		}

		var event *GitHubPullRequestRequest
		if err := json.Unmarshal(body, &event); err != nil || event == nil {
			http.Error(w, "invalid pull_request payload", http.StatusBadRequest)
			return
		}

		var repository string
		if event.Repository != nil {
			repository = event.Repository.FullName
		}
		log.Printf("github webhook event=pull_request delivery=%q action=%q repository=%q number=%d",
			r.Header.Get("X-GitHub-Delivery"), event.Action, repository, event.Number)
		fmt.Fprint(w, "OK\n")
	})
	return mux
}

func main() {

	env := os.Getenv("env")
	if env == "" {
		panic("env is empty")
	}
	err := godotenv.Load(fmt.Sprintf(".env.%s", env))
	if err != nil {
		os.Exit(1)
	}

	secrets, err := app_utils.NewSecretsManager(os.Getenv(SECRET_PASSWORD_FILE))
	if err != nil {
		log.Fatalf("initialize secrets manager: %v", err)
	}

	secretItem, configured := os.LookupEnv("GITHUB_WEBHOOK_SECRET_ITEM")
	if !configured {
		secretItem = defaultWebhookSecretItem
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	webhookSecret, err := secrets.GetSecret(ctx, secretItem)
	cancel()
	if err != nil {
		log.Fatalf("load GitHub webhook secret from Vaultwarden: %v", err)
	}
	if strings.TrimSpace(webhookSecret) == "" {
		log.Fatal("GitHub webhook secret from Vaultwarden is empty")
	}

	port := "8888"
	if value, ok := os.LookupEnv("PORT"); ok {
		if _, err := strconv.ParseUint(value, 10, 16); err != nil {
			log.Fatalf("invalid PORT %q: %v", value, err)
		}
		port = value
	}

	address := net.JoinHostPort("127.0.0.1", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("review-bot-api listening on %s", listener.Addr())
	log.Fatal(http.Serve(listener, handler(webhookSecret)))
}
