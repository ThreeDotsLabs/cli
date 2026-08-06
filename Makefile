.PHONY: proto
proto:
	protoc \
    		--proto_path=trainings/api/protobuf trainings/api/protobuf/server.proto \
    		--go_out=trainings/genproto --go_opt=paths=source_relative \
    		--go-grpc_opt=require_unimplemented_servers=false \
    		--go-grpc_out=trainings/genproto --go-grpc_opt=paths=source_relative \

.PHONY: install
install:
	go install ./tdl/


.PHONY: lint
lint:
	golangci-lint run

.PHONY: test
test:
	go test ./...

.PHONY: fmt
fmt:
	goimports -local github.com/ThreeDotsLabs -l -w .


.PHONY: update-nix-hash
update-nix-hash:
	@echo "Resetting vendorHash in Nix files..."
	@sed -i.bak 's/vendorHash = ".*"/vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="/' flake.nix default.nix
	@rm -f flake.nix.bak default.nix.bak
	@echo "Calculating new vendorHash..."
	@NEW_HASH=$$(nix build 2>&1 | awk '/got:/{print $$2}' | head -n 1) && \
	if [ -n "$$NEW_HASH" ]; then \
		echo "Found new hash: $$NEW_HASH"; \
		sed -i.bak "s/vendorHash = \".*\"/vendorHash = \"$$NEW_HASH\"/" flake.nix default.nix; \
		rm -f flake.nix.bak default.nix.bak; \
		echo "Nix files updated successfully."; \
	else \
		echo "Failed to calculate new hash. Are there other Nix build errors?"; \
		git checkout flake.nix default.nix; \
		exit 1; \
	fi
