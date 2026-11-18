package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	app_utils "github.com/kumarmo2/patchguard/review-bot-api/utils"
)

const CLOUDFLARE_SECRET_NAME string = "CLOUDFLARE_SECRET_NAME"
const SECRET_PASSWORD_FILE string = "SECRET_PASSWORD_FILE"

func handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /webhooks/github", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-GitHub-Event") != "pull_request" {
			fmt.Fprint(w, "OK\n")
			return
		}

		const maxWebhookBodyBytes = 25_000_000
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

		// TODO: Verify X-Hub-Signature-256 over body before decoding it.
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

	_, err = app_utils.NewSecretsManager(os.Getenv(SECRET_PASSWORD_FILE))

	if err != nil {
		os.Exit(1)
		return
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
	log.Fatal(http.Serve(listener, handler()))
}
