BINARY_NAME=home_be
GOOS=linux
BUILDARCH ?= $(shell uname -m)

ifeq ($(BUILDARCH),aarch64)
	BUILDARCH=arm64
endif
ifeq ($(BUILDARCH),x86_64)
	BUILDARCH=amd64
endif

dep:
	go mod tidy && go mod vendor && go fmt

build: clean
	go mod init github.com/janevala/home_be
	go mod tidy
	go get github.com/mmcdole/gofeed
	go get github.com/google/uuid
	go get github.com/lib/pq
	# go get github.com/modelcontextprotocol/go-sdk/mcp
	go get github.com/rifaideen/talkative
	go get github.com/graphql-go/graphql

debug: build
	GOARCH=${BUILDARCH} go build -tags debug -o ${BINARY_NAME}_${BUILDARCH} main.go

release: build
	GOARCH=${BUILDARCH} go build -tags release -o ${BINARY_NAME}_${BUILDARCH} main.go

run:
	./${BINARY_NAME}_${BUILDARCH}

clean:
	go clean
	go clean -cache
	rm -rf vendor
	rm -rf go.sum
	rm -rf go.mod
	rm -f ${BINARY_NAME}_${BUILDARCH}

rebuild: clean build

help:
	@echo "Available targets:"
	@echo "  dep       - Install dependencies"
	@echo "  build     - Build mods"
	@echo "  debug     - Build debug version"
	@echo "  release   - Build release version"
	@echo "  run       - Run the application"
	@echo "  clean     - Clean up the build directory"
	@echo "  rebuild   - Rebuild the application"
	@echo "  help      - Show this help message"
