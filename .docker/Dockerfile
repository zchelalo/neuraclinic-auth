FROM golang:1.25.5-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.3
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -trimpath -ldflags="-s -w" -o /out/neuraclinic-auth ./cmd

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /out/neuraclinic-auth /usr/local/bin/neuraclinic-auth
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate
COPY internal/db/migrations /app/migrations
COPY scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
COPY scripts/run-migrations.sh /usr/local/bin/run-migrations.sh

RUN chmod +x /usr/local/bin/docker-entrypoint.sh /usr/local/bin/run-migrations.sh

ENV ENVIRONMENT=production

EXPOSE 8000

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["neuraclinic-auth"]
