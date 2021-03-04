
.PHONY: all
all: proto
	go build

.PHONY: proto
proto:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/kvstore.proto
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/raft.proto

.PHONY: clean
clean:
	rm -rf proto/*.pb.go

.PHONY: openapi-stub
openapi-stub:
	docker run --rm -v "${PWD}:/local" openapitools/openapi-generator-cli generate -i /local/web/brokerd.yaml -g go -o /local/apiclient

.PHONY: openapi-skeleton
openapi-skeleton:
	@docker run --rm -v "${PWD}:/local" \
	openapitools/openapi-generator-cli generate -g go-gin-server \
	-i /local/web/brokerd-oas3.yaml -o /local/web/openapi	
	@gofmt -w web/openapi/go