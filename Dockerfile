FROM golang:1.22.1-bookworm as builder

WORKDIR /app

COPY go.mod go.sum /app/
RUN go mod download

COPY main.go /app/
COPY ./internal /app/internal
RUN CGO_ENABLED=0 go build .

FROM alpine:3.19

WORKDIR /app
COPY --from=builder /app/jaeger-postgresql .
COPY ./hack/run.sh /app/run.sh
RUN chmod +x /app/run.sh

CMD ["./run.sh"]
