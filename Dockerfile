# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /out/webhook2stdout .

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=builder /out/webhook2stdout /app/webhook2stdout
COPY config.yaml /app/config.yaml

EXPOSE 8080

ENTRYPOINT ["/app/webhook2stdout"]
CMD ["-config", "/app/config.yaml"]
