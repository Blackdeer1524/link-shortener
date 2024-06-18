FROM golang:1.22 as builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY ./cmd/authenticator/authenticator.go ./cmd/authenticator/authenticator.go
COPY ./pkg/ ./pkg/

ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target=/go/pkg/mod/ go build -v -o authenticator ./cmd/authenticator/authenticator.go 

EXPOSE 8080
ENTRYPOINT ["./authenticator"]
