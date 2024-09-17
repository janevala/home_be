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
	@echo "make dep - Install dependencies for compiling the binary file"
	@echo "make build - Build the binary file"
	@echo "make run - Run the binary file"
	@echo "make build_and_run - Build and run the binary file"
	@echo "make clean - Remove the binary file"

dep:
	go mod tidy && go mod vendor && go fmt

build:
	go mod init github.com/janevala/home_be
	go mod tidy
	go get github.com/mmcdole/gofeed
	go get github.com/google/uuid
	go get github.com/lib/pq
	GOARCH=${BUILDARCH} go build -o ${BINARY_NAME}_${BUILDARCH} main.go

run:
	./${BINARY_NAME}_${BUILDARCH}

clean:
	go clean
	rm -rf vendor
	rm -rf go.sum
	rm -rf go.mod
	rm -f ${BINARY_NAME}_${BUILDARCH}
