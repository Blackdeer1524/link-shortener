FROM golang:1.22 as build

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY ./cmd/authenticator/authenticator.go ./cmd/authenticator/authenticator.go
COPY ./pkg/ ./pkg/

ENV CGO_ENABLED=0
ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target=/root/.cache/go-build go build -v -o /usr/local/bin/authenticator ./cmd/authenticator/authenticator.go 

FROM alpine:3.14 as runner
COPY --from=build /usr/local/bin/authenticator /usr/local/bin/authenticator
EXPOSE 8080

ENTRYPOINT ["authenticator"]
