DBSTRING=postgres://postgres:password@localhost:5432/jaeger
JAEGER_VERSION=1.54.0

.PHONY: dev
dev:
	go build .
	SPAN_STORAGE_TYPE="grpc-plugin" GRPC_STORAGE_PLUGIN_BINARY=./jaeger-postgresql GRPC_STORAGE_PLUGIN_CONFIGURATION_FILE=./hack/config.yaml ./hack/jaeger-all-in-one 

.PHONY: install-all-in-one
install-all-in-one:
	wget https://github.com/jaegertracing/jaeger/releases/download/v$(JAEGER_VERSION)/jaeger-$(JAEGER_VERSION)-linux-amd64.tar.gz -P ./hack/
	tar  -C ./hack --extract --file ./hack/jaeger-$(JAEGER_VERSION)-linux-amd64.tar.gz jaeger-$(JAEGER_VERSION)-linux-amd64/jaeger-all-in-one
	rm ./hack/jaeger-$(JAEGER_VERSION)-linux-amd64.tar.gz
	mv ./hack/jaeger-$(JAEGER_VERSION)-linux-amd64/jaeger-all-in-one ./hack/jaeger-all-in-one
	rmdir ./hack/jaeger-$(JAEGER_VERSION)-linux-amd64/	

.PHONY: generate
generate: 
	sqlc generate
	
# Install DB migration tool
.PHONY: install-goose
install-goose:
	@sh -c "which goose > /dev/null || go install github.com/pressly/goose/v3/cmd/goose@latest"

.PHONY: install-sqlc
install-sqlc: 
	@sh -c "which sqlc > /dev/null || go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest"

# Migrate SQL schema to latest version
.PHONY: migrate
migrate: install-goose
	GOOSE_DBSTRING=$(DBSTRING) GOOSE_DRIVER=postgres goose -dir ./internal/sql/migrations up

# Redo SQL schema migration
.PHONY: migrate-redo
migrate-redo: install-goose
	GOOSE_DBSTRING=$(DBSTRING) GOOSE_DRIVER=postgres goose -dir ./internal/sql/migrations redo

# Rollback SQL schema by one version
.PHONY: migrate-down
migrate-down: install-goose
	GOOSE_DBSTRING=$(DBSTRING) GOOSE_DRIVER=postgres goose -dir ./internal/sql/migrations down

# Get SQL schema migration status
.PHONY: migrate-status
migrate-status: install-goose
	GOOSE_DBSTRING=$(DBSTRING) GOOSE_DRIVER=postgres goose -dir ./internal/sql/migrations status