PKGNAME=golicenser
GOFMT_FILES?=$$(find . -name '*.go')

default: build

gitsha := $(shell git log -n1 --pretty='%h')
version=$(shell git describe --exact-match --tags "$(gitsha)" 2>/dev/null)
ifeq ($(version),)
	version := $(gitsha)
endif
ldflags=-ldflags='-X main.version=$(version)'
.PHONY: build
build:
	@echo "==> Building $(PKGNAME)"
	go build $(ldflags) -o out/$(PKGNAME) ./...

.PHONY: clean
clean:
	@echo "==> Removing previously compiled binaries"
	@rm -rf out

.PHONY: fmt
fmt:
	gofmt -s -w $(GOFMT_FILES)
	goimports -w $(GOFMT_FILES)

.PHONY: install
install: build
	@echo "==> Installing $(PKGNAME)"
	@go install

.PHONY: lint
lint:
	@echo "==> Linting all packages..."
	golangci-lint run ./... -E gofmt -E golint

.PHONY: setup
setup:
	@echo "==> Installing linter..."
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.21.0

	@echo "==> Installing import checker..."
	go get golang.org/x/tools/cmd/goimports

	@echo "==> Linking git hooks..."
	find .git/hooks -type l -exec rm {} \;
	find .githooks -type f -exec ln -sf ../../{} .git/hooks/ \;

.PHONY: uninstall
uninstall: build
	@echo "==> Removing $(GOPATH)/bin/$(PKGNAME)"
	@rm -f $(GOPATH)/bin/$(PKGNAME)
