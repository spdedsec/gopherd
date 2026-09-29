APP=gopherd

.PHONY: run build test vet fmt lint docker-up docker-down

run:
	go run ./cmd/gopherd

build:
	go build -trimpath -o bin/$(APP) ./cmd/gopherd

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

docker-up:
	docker compose up --build -d db app

docker-down:
	docker compose down
