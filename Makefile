BINARY_NAME=home_be

GOARCH_AMD64=amd64
GOARCH_ARM64=arm64
GOARCH=${GOARCH_AMD64}

dep:
	go mod tidy && go mod vendor && go fmt

build:
	 go build -o ${BINARY_NAME}_${GOARCH} main.go

build_linux:
	GOOS=linux GOARCH=${GOARCH_AMD64} go build -o ${BINARY_NAME}_${GOARCH_AMD64} main.go

build_arm:
	GOOS=linux GOARCH=${GOARCH_ARM64} go build -o ${BINARY_NAME}_${GOARCH_ARM64} main.go

run:
	go run main.go

run_linux:
	./${BINARY_NAME}_${GOARCH_AMD64}

run_arm:
	./${BINARY_NAME}_${GOARCH_ARM64}

run_production:
	./${BINARY_NAME}_${GOARCH_ARM64}

clean:
	go clean
	rm -rf vendor
	rm -f ${BINARY_NAME}_${GOARCH_AMD64}
	rm -f ${BINARY_NAME}_${GOARCH_ARM64}

