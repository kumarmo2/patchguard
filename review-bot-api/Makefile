BINARY := target/review-bot-api

.PHONY: build run test clean deploy-ready deploy

build:
	mkdir -p target
	go build -o $(BINARY) .

run: build
	exec ./$(BINARY)

test:
	go test ./...

clean:
	rm -f $(BINARY)

deploy-ready:
	sh ./deploy-ready.sh

deploy:
	sh ./deploy.sh
