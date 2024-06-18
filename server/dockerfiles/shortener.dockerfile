FROM golang:1.22 as build

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY ./cmd/shortener/shortener.go ./cmd/shortener/shortener.go
COPY ./pkg/ ./pkg/

ENV CGO_ENABLED=0
ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target=/root/.cache/go-build go build -v -o /usr/local/bin/shortener ./cmd/shortener/shortener.go 

FROM alpine:3.14 as runner
COPY --from=build /usr/local/bin/shortener /usr/local/bin/shortener
EXPOSE 8080

ENTRYPOINT ["shortener"]

