# Stage 1: Build
FROM golang:1.22 AS builder

WORKDIR /app

COPY . .

RUN go mod tidy
RUN go build -o /app/siggen main.go

ENTRYPOINT ["/app/siggen"]
