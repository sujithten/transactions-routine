FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY . .

RUN go mod tidy

RUN go install github.com/swaggo/swag/cmd/swag@v1.16.3

RUN swag init -g cmd/api/main.go --output docs

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api


FROM alpine:3.19

COPY --from=builder /bin/api /bin/api

EXPOSE 8080

ENTRYPOINT ["/bin/api"]
