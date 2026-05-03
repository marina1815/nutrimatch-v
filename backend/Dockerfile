FROM golang:1.26.2-trixie AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.27.0

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/nutrimatch-api ./cmd/api
RUN cp /go/bin/goose /out/goose

FROM debian:trixie-slim AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=build /out/nutrimatch-api /usr/local/bin/nutrimatch-api
COPY --from=build /out/goose /usr/local/bin/goose
COPY --from=build /src/migrations /app/migrations

ENV APP_PORT=8080
EXPOSE 8080

CMD ["nutrimatch-api"]
