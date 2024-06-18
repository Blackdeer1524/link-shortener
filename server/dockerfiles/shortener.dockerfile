FROM golang:1.22 as builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY ./cmd/shortener/shortener.go ./cmd/shortener/shortener.go
COPY ./pkg/ ./pkg/

ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target=/root/.cache/go-build go build -v -o shortener ./cmd/shortener/shortener.go 

EXPOSE 8080
ENTRYPOINT ["./shortener"]
