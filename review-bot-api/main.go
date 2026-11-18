package main

import (
	"fmt"
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
		eventType := r.Header.Get("X-GitHub-Event")
		fmt.Println("eventType: ", eventType)
		fmt.Fprint(w, "OK\n")
		// TODO: first this we need to do is authroize the request.
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
