.PHONY: all build test lint fmt check-fmt clean coverage vet ast staticcheck
.DEFAULT_GOAL := all

test:
	@go test -v ./...

lint:
	@golangci-lint run

fmt:
	@gofmt -s -w .

check-fmt:
	@gofmt -l . | tee /dev/stderr | grep -q '^' && echo "Code is not formatted" && exit 1 || echo "Code is formatted"

coverage:
	@mkdir -p coverage
	@go test -coverprofile=coverage/coverage.out ./... && \
	go tool cover -func=coverage/coverage.out && \
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html; \
	open coverage/coverage.html

vet:
	@go vet ./...

staticcheck:
	@staticcheck ./...

ast:
	@gosec . ./internal/...

docs:
	@echo $$(sleep 2 && open http://localhost:6060/pkg/github.com/ren3gadem4rm0t/imperva-export-cli/internal/cmd/) &
	@godoc -play -http localhost:6060 -v

clean:
	@go clean
	@rm -f ./coverage.out ./coverage.html coverage/coverage.out coverage/coverage.html
	@rm -rf ./coverage

fuzz:
	@echo "Running fuzz tests..."

build:
	@go build -o imperva-export-cli .

all: test lint fmt vet staticcheck ast
