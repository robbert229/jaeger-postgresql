FROM golang:1.22 AS build

WORKDIR /build

COPY . .

RUN go mod tidy && \
    go build -o /jaeger-postgresql ./cmd/jaeger-postgresql/ && \
    go build -o /jaeger-postgresql-cleaner ./cmd/jaeger-postgresql-cleaner/

FROM busybox AS runner

WORKDIR /app

COPY --from=build /jaeger-postgresql /jaeger-postgresql
COPY --from=build /jaeger-postgresql-cleaner /jaeger-postgresql-cleaner

CMD ["/jaeger-postgresql"]
