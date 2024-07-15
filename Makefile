
dep:
	go mod tidy && go mod vendor && go fmt

run:
	go run main.go

clean:
	go clean
	rm -rf vendor