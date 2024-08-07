BINARY_NAME=home_be

BUILDARCH ?= $(shell uname -m)

ifeq ($(BUILDARCH),aarch64)
	BUILDARCH=arm64
endif
ifeq ($(BUILDARCH),x86_64)
	BUILDARCH=amd64
endif

dep:
	go mod tidy && go mod vendor && go fmt

build:
	GOOS=linux GOARCH=${BUILDARCH} go build -o ${BINARY_NAME}_${BUILDARCH} main.go

run:
	./${BINARY_NAME}_${BUILDARCH}

build_and_run:
	GOOS=linux GOARCH=${BUILDARCH} go build -o ${BINARY_NAME}_${BUILDARCH} main.go && ./${BINARY_NAME}_${BUILDARCH}

clean:
	go clean
	rm -rf vendor
	rm -f ${BINARY_NAME}_${BUILDARCH}
