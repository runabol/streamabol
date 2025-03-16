FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o streamabol

FROM alpine:latest
RUN apk add --no-cache ffmpeg
WORKDIR /app
COPY --from=builder /app/streamabol .
EXPOSE 8080
CMD ["./streamabol"]