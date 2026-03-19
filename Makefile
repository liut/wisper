.SILENT :
.PHONY : all clean dist lint test

NAME:=webpawm
ROOF:=github.com/liut/$(NAME)

# Get version: use tag if available, otherwise use short hash
TAG := $(shell git describe --tags --always --long 2>/dev/null || git rev-parse --short HEAD)
VERSION := $(or $(VERSION),$(TAG))
LDFLAGS:=-X main.version=$(VERSION)

GO=$(shell which go)
GOMOD=$(shell echo "$${GO111MODULE:-auto}")

main:
	mkdir -p dist
	GO111MODULE=$(GOMOD) $(GO) build -ldflags "$(LDFLAGS) -s -w" -o dist/$(NAME) .

all: dist

dist: dist/linux_amd64/$(NAME) dist/darwin_amd64/$(NAME) dist/darwin_arm64/$(NAME) dist/windows_amd64/$(NAME).exe

dist/linux_amd64/$(NAME):
	mkdir -p dist/linux_amd64
	GO111MODULE=$(GOMOD) GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS) -s -w" -o $@ .

dist/darwin_amd64/$(NAME):
	mkdir -p dist/darwin_amd64
	GO111MODULE=$(GOMOD) GOOS=darwin GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS) -s -w" -o $@ .

dist/darwin_arm64/$(NAME):
	mkdir -p dist/darwin_arm64
	GO111MODULE=$(GOMOD) GOOS=darwin GOARCH=arm64 $(GO) build -ldflags "$(LDFLAGS) -s -w" -o $@ .

dist/windows_amd64/$(NAME).exe:
	mkdir -p dist/windows_amd64
	GO111MODULE=$(GOMOD) GOOS=windows GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS) -s -w" -o $@ .

package: dist/linux_amd64/$(NAME)
	tar -cvJf $(NAME)-linux-amd64-$(TAG).tar.xz -C dist/linux_amd64 $(NAME)

clean:
	rm -rf dist
	rm -f $(NAME) $(NAME)-*

lint:
	GO111MODULE=$(GOMOD) golangci-lint run -v ./...

test:
	GO111MODULE=$(GOMOD) $(GO) test -v ./...
