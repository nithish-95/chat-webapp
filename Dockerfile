FROM golang:alpine AS builder

RUN mkdir -p /app/bin
WORKDIR /app

COPY . .

RUN go build -o /app/bin/chatapp .

FROM alpine:latest

RUN addgroup -S appgroup
RUN adduser -S appuser -G appgroup

RUN mkdir /app
WORKDIR /app

COPY --from=builder /app/bin/chatapp .
COPY --from=builder /app/*.html .

RUN chown -R appuser:appgroup .
USER appuser

ENTRYPOINT ["./chatapp"]

EXPOSE 3000