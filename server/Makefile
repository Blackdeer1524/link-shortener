.PHONY: proto

proto: 
	protoc --go_out=. --go_opt=paths=source_relative \
           --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       proto/blackbox/blackbox.proto

test: proto
	go mod download && go mod verify
	go clean -testcache
	mockery
	go test ./internal/...
