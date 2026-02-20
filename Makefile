# Any args passed to the make script, use with $(call args, default_value)
# args = `arg="$(filter-out $@,$(MAKECMDGOALS))" && echo $${arg:-${1}}`

GOOS ?= linux
BUILDARCH ?= $(shell uname -m)
BINARY_NAME := home_be
VERSION := $(shell git describe --always --long --dirty)
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/)

ifeq ($(BUILDARCH),aarch64)
	BUILDARCH=arm64
endif
ifeq ($(BUILDARCH),x86_64)
	BUILDARCH=amd64
endif

help:
	@echo "Available targets:"
	@echo "  vet       - Run go vet on the codebase"
	@echo "  dep       - Install dependencies"
	@echo "  build     - Build mods"
	@echo "  debug     - Build debug version"
	@echo "  release   - Build release version"
	@echo "  run       - Run the application"
	@echo "  clean     - Clean up the build directory"
	@echo "  rebuild   - Rebuild the application"
	@echo "  help      - Show this help message"

# test:
# 	@go test -short ${PKG_LIST}

# lint:
# 	@for file in ${GO_FILES} ;  do \
# 		golint $$file ; \
# 	done

dep:
	go mod vendor && go fmt

vet: clean
	go mod init github.com/janevala/home_be
	go vet ${PKG_LIST}

build: clean
	go mod init github.com/janevala/home_be
	go mod tidy
	go get github.com/mmcdole/gofeed
	go get github.com/google/uuid
	go get github.com/lib/pq
	go get github.com/rifaideen/talkative
	go get github.com/joho/godotenv

debug: build
	GOARCH=${BUILDARCH} go build -v -tags debug -o ${BINARY_NAME}_${BUILDARCH} -ldflags="-X main.version=${VERSION}" main.go

release: build
	GOARCH=${BUILDARCH} go build -v -tags release -o ${BINARY_NAME}_${BUILDARCH} -ldflags="-X main.version=${VERSION}" main.go

run:
	go run -tags debug main.go

clean:
	go clean
	go clean -cache
	go clean -modcache
	rm -rf vendor
	rm -rf go.sum
	rm -rf go.mod
	rm -f ${BINARY_NAME}_${BUILDARCH}

rebuild: clean build
