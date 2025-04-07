FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main .

# 运行应用
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/render/template /root/render/template 
COPY --from=builder /app/main .
EXPOSE 30001
CMD ["./main"]
