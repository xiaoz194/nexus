.PHONY: run build tidy fmt clean

run:
	go run ./cmd/server

build:
	go build ./...

tidy:
	go mod tidy

fmt:
	gofmt -w .

clean:
	rm -f nexus.db
