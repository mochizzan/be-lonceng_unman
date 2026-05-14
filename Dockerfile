FROM golang:1.26-alpine AS builder

WORKDIR /app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -v -o api ./cmd/api

FROM golang:1.26-alpine

WORKDIR /app

COPY --from=builder /app/api .

EXPOSE 8080

CMD ["./api"]
