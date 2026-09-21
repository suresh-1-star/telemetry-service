FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download || true
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o telemetry-api main.go

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/telemetry-api .
EXPOSE 8080
CMD ["./telemetry-api"]