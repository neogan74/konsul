FROM golang:1.24-alpine

WORKDIR /app

COPY . .

RUN go build -o konsul

EXPOSE 8888

CMD ["./konsul"]