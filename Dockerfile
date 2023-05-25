FROM golang:1.20-alpine as builder

WORKDIR /src
COPY go.sum go.sum
COPY go.mod go.mod
RUN go mod download
COPY main.go main.go
RUN go build -o knada-ping .

CMD ["/src/knada-ping"]