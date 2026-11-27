# syntax=docker/dockerfile:1
FROM golang:1.27.3-alpine3.24 AS build

WORKDIR /src
ENV CGO_ENABLED=0 GOTOOLCHAIN=local

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY auth-microservice ./auth-microservice
COPY internal ./internal
COPY pkg ./pkg
COPY db/migrations ./db/migrations
RUN go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/app
RUN go build -trimpath -ldflags="-s -w" -o /out/auth ./auth-microservice/cmd/auth

FROM alpine:3.24 AS runtime
RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app \
    && mkdir -p /data/media \
    && chown -R app:app /data/media

WORKDIR /app
COPY --from=build /out/ ./
RUN mkdir -p /app/media && chown app:app /app/media
USER app:app

EXPOSE 8080
ENTRYPOINT ["/app/app"]
CMD ["server"]
