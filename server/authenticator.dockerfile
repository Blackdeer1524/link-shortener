FROM golang:1.22 as build

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY ./cmd/authenticator/authenticator.go ./cmd/authenticator/authenticator.go
COPY ./pkg/ ./pkg/

RUN go build -v -o /usr/local/bin/authenticator ./cmd/authenticator/authenticator.go

EXPOSE 8080

CMD ["authenticator"]
