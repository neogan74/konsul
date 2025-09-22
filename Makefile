APP_NAME=konsul

.PHONY: build run test clean docker-build docker-run

build:
	go build -o $(APP_NAME) cmd/konsul/main.go

run:
	go run cmd/konsul/main.go

test:
	go test -v ./...

clean:
	rm -f $(APP_NAME)

docker-build:
	docker build -t $(APP_NAME):latest .

docker-run:
	docker run --rm -p 8888:8888 $(APP_NAME):latest
air:
	air --build.cmd "go build -o bin/konsul cmd/konsul/main.go" --build.bin "./bin/konsul"