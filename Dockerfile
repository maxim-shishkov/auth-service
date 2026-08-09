FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
COPY config.json /config.json
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY config.json /config.json
COPY --from=builder /out/server /usr/local/bin/server
COPY --from=builder /src/migrations /migrations

EXPOSE 8080

CMD ["/usr/local/bin/server"]
