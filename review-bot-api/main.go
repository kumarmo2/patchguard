package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
)

func handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /webhooks/github", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("inside the webhook")
		fmt.Fprint(w, "OK\n")
		// TODO: first this we need to do is authroize the request.
	})
	return mux
}

func main() {
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
