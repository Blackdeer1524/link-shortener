FROM golang:1.22 as build

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY ./cmd/shortener/shortener.go ./cmd/shortener/shortener.go
COPY ./pkg/ ./pkg/

RUN go build -v -o /usr/local/bin/shortener ./cmd/shortener/shortener.go

EXPOSE 8080

CMD ["shortener"]
