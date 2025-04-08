APP_NAME=konsul

.PHONY: build run test clean docker-build docker-run

build:
	go build -o $(APP_NAME)

run:
	go run main.go

test:
	go test -v ./...

clean:
	rm -f $(APP_NAME)

docker-build:
	docker build -t $(APP_NAME):latest .

docker-run:
	docker run --rm -p 8080:8080 $(APP_NAME):latest
