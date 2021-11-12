.PHONY: proto
proto:
	protoc \
    		--proto_path=trainings/api/protobuf trainings/api/protobuf/server.proto \
    		--go_out=trainings/genproto --go_opt=paths=source_relative \
    		--go-grpc_opt=require_unimplemented_servers=false \
    		--go-grpc_out=trainings/genproto --go-grpc_opt=paths=source_relative \

.PHONY: install
install:
	go install .


.PHONY: lint
lint:
	golangci-lint run

.PHONY: test
test:
	go test ./...

.PHONY: fmt
fmt:
	goimports -local github.com/ThreeDotsLabs -l -w .

