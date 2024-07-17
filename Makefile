BINARY_NAME=home_be
GOARCH=amd64

dep:
	go mod tidy && go mod vendor && go fmt

run:
	go run main.go

build:
	 go build -o ${BINARY_NAME}_${GOARCH} main.go

clean:
	go clean
	rm -rf vendor
	rm -f ${BINARY_NAME}_${GOARCH}
