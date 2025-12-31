BINARY_NAME=tri
OS=linux

build:
	GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -o bin/${BINARY_NAME}-${OS}-amd64 main.go
	GOARCH=arm64 GOOS=linux CGO_ENABLED=0 go build -o bin/${BINARY_NAME}-${OS}-arm64 main.go
docker:
	docker build --target=release -t autovia/tri:0.0.1 .
