BINARY_NAME=home_be
GOOS=linux
BUILDARCH ?= $(shell uname -m)

ifeq ($(BUILDARCH),aarch64)
	BUILDARCH=arm64
endif
ifeq ($(BUILDARCH),x86_64)
	BUILDARCH=amd64
endif

help:
	@echo "make clean - Remove the binary file"
	@echo "make build - Build the binary file"
	@echo "make dep - Install dependencies for compiling the binary file"
	@echo "make run - Run the binary file"
	@echo
	@echo "Run all make commands at once:"
	@echo "make clean && make build && make dep && make run"

dep:
	go mod tidy && go mod vendor && go fmt

build:
	go mod init github.com/janevala/home_be
	go mod tidy
	go get github.com/mmcdole/gofeed
	go get github.com/google/uuid
	go get github.com/lib/pq
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

rebuild: clean debug
