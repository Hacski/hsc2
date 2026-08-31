FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /hsc2-server ./cmd/server

FROM alpine:3.20

RUN addgroup -S hsc2 && adduser -S hsc2 -G hsc2

WORKDIR /app

COPY --from=builder /hsc2-server .

RUN mkdir -p /data/db /data/certs && chown -R hsc2:hsc2 /data

USER hsc2

VOLUME ["/data/db", "/data/certs"]

EXPOSE 8443 4444 5353

ENTRYPOINT ["./hsc2-server"]
