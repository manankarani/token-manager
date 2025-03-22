# Stage 1: Build the Go application
FROM golang:1.23-alpine3.19 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN apk update && apk add --no-cache ca-certificates tzdata && update-ca-certificates

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o token-server ./cmd/main.go

# Stage 2: Minimal runtime container
FROM scratch as final
COPY ./env/config/ ./env/config/
COPY --from=builder /app/token-server .
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENV TZ Asia/Kolkata


EXPOSE 8080
CMD ["./token-server"]
