FROM golang:1.20-alpine AS builder

RUN apk --no-cache add build-base

WORKDIR /app

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o goodnews

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/goodnews .

RUN chmod +x /app/goodnews

RUN mkdir /app/data

ENTRYPOINT ["./goodnews"]
