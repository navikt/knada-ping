FROM golang:1.21-alpine as builder

WORKDIR /src

COPY go.sum go.sum
COPY go.mod go.mod
RUN go mod download

COPY main.go main.go
RUN go build -o knada-ping .

FROM cgr.dev/chainguard/static:latest

WORKDIR /app
COPY --from=builder /src/knada-ping /app/knada-ping

CMD ["/app/knada-ping"]
