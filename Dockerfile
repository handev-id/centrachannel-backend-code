FROM golang:1.26.2-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server/main.go
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest


FROM alpine:latest

RUN apk --no-cache add ca-certificates postgresql-client

WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate
COPY --from=builder /app/database/migrations ./database/migrations
COPY --from=builder /app/database/seed.sql ./database/seed.sql

COPY entrypoint.sh .

EXPOSE 3000

CMD ["./entrypoint.sh"]
