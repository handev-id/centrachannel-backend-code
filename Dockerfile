FROM golang:1.26.2-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/centrachannel ./cmd/server/main.go
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.0

FROM alpine:3.22

RUN apk --no-cache add ca-certificates wget \
    && addgroup -S app \
    && adduser -S -G app app

WORKDIR /app

COPY --from=builder /out/centrachannel ./centrachannel
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate
COPY --from=builder /app/database/migrations ./database/migrations
COPY --from=builder /app/database/seed.sql ./database/seed.sql
COPY --chown=app:app entrypoint.sh ./entrypoint.sh

RUN chmod 0755 ./entrypoint.sh ./centrachannel /usr/local/bin/migrate

USER app
EXPOSE 4001

ENTRYPOINT ["./entrypoint.sh"]
