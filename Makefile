
default: build

build: fmt lint
	@go build -ldflags="-s -w"

fmt:
	@goimports -w -l .
	@gofumpt -w -l .

lint:
	@go vet ./...

nowayland: fmt lint
	@go build -ldflags="-s -w" -tags nowayland
