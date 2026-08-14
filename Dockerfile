FROM golang:1.26.6 AS builder

WORKDIR /Sclera

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main ./cmd/server

FROM debian:bookworm-slim

WORKDIR /Sclera

COPY --from=builder /Sclera/main .

CMD ["./main"]