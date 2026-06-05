FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
COPY vendor/ vendor/
COPY . .

RUN go build -mod=vendor -o server .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/server .
COPY frontend/ frontend/

EXPOSE 8080
CMD ["./server"]
