
default: build

release: fmt lint
	@go build -ldflags="-s -w" --tags nowayland

build: fmt lint
	@go build -tags nowayland

fmt:
	@goimports -w -l .
	@gofumpt -w -l .

lint:
	@go vet ./...

yeswayland: fmt lint
	@go build
