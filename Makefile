
default: build

build: fmt lint
	@go build -tags nowayland

fmt:
	@goimports -w -l .
	@gofumpt -w -l .

lint:
	@go vet ./...

release: fmt lint
	@go build -ldflags="-s -w" --tags nowayland

wayland: fmt lint
	@go build
