# syntax=docker/dockerfile:1

FROM golang:1.23 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /leveltalk ./cmd/server

FROM gcr.io/distroless/base-debian12
WORKDIR /srv/leveltalk

COPY --from=builder /leveltalk /usr/local/bin/leveltalk

ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/leveltalk"]


