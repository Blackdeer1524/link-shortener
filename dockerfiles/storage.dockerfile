FROM golang:1.22-bookworm as build

RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
    --mount=target=/var/cache/apt,type=cache,sharing=locked \
    rm -f /etc/apt/apt.conf.d/docker-clean \
    && apt update \
    && apt -y --no-install-recommends install \
        protobuf-compiler

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY ./cmd/storage/storage.go ./cmd/storage/storage.go
COPY ./pkg/ ./pkg/
COPY ./internal/storage ./internal/storage

COPY ./proto/blackbox/blackbox.proto ./proto/blackbox/blackbox.proto 
RUN protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/blackbox/blackbox.proto

ENV CGO_ENABLED=0
ENV GOCACHE=/root/.cache/go-build 
RUN --mount=type=cache,target=/root/.cache/go-build go build -v -o /usr/local/bin/storage ./cmd/storage/storage.go 

FROM alpine:3.14 as runner
COPY --from=build /usr/local/bin/storage /usr/local/bin/storage
EXPOSE 8080

ENTRYPOINT ["storage"]
