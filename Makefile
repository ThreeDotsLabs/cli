.PHONY: proto
proto:
	protoc \
    		--proto_path=course/api/protobuf course/api/protobuf/server.proto \
    		--go_out=course/genproto --go_opt=paths=source_relative \
    		--go-grpc_opt=require_unimplemented_servers=false \
    		--go-grpc_out=course/genproto --go-grpc_opt=paths=source_relative \

.PHONY: install
install:
	go install .