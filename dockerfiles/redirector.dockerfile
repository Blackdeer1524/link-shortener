FROM golang:1.22-bookworm as build

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY ./cmd/redirector/redirector.go ./cmd/redirector/redirector.go
COPY ./pkg/ ./pkg/
COPY ./internal/redirector ./internal/redirector

ENV CGO_ENABLED=0
ENV GOCACHE=/root/.cache/go-build 
RUN --mount=type=cache,target=/root/.cache/go-build go build -v -o /usr/local/bin/redirector ./cmd/redirector/redirector.go 

FROM alpine:3.14 as runner
COPY --from=build /usr/local/bin/redirector /usr/local/bin/redirector
EXPOSE 8080

ENTRYPOINT ["redirector"]

