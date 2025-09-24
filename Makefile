APP_NAME=konsul
CLI_NAME=konsulctl

.PHONY: build run test clean docker-build docker-run build-cli

build:
	go build -o $(APP_NAME) cmd/konsul/main.go

build-cli:
	go build -o $(CLI_NAME) ./cmd/konsulctl

run:
	go run cmd/konsul/main.go

test:
	go test -v ./...

clean:
	rm -f $(APP_NAME) $(CLI_NAME)

docker-build:
	docker build -t $(APP_NAME):latest .

docker-run:
	docker run --rm -p 8888:8888 $(APP_NAME):latest
air:
	air --build.cmd "go build -o bin/konsul cmd/konsul/main.go" --build.bin "./bin/konsul"