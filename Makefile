PKGNAME=golicenser

default: build

clean:
	@echo "==> Removing previously compiled binaries"
	@rm -rf out

build: clean
	@echo "==> Building $(PKGNAME)"
	@go build -o out/$(PKGNAME)

install: build
	@echo "==> Installing $(PKGNAME)"
	@go install

uninstall: build
	@echo "==> Removing $(GOPATH)/bin/$(PKGNAME)"
	@rm -f $(GOPATH)/bin/$(PKGNAME)

.PHONY: default clean build install uninstall
