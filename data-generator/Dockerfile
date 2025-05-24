FROM golang:1.22.4 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o data-generator main.go

# Minimal runtime image
FROM gcr.io/distroless/static:nonroot

WORKDIR /app
COPY --from=builder /app/data-generator .
ENTRYPOINT ["/app/data-generator"]