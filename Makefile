DBSTRING = postgres://postgres:password@localhost:5432/jaeger
JAEGER_VERSION = 1.54.0
VERSION ?= 'v1.5.0' # x-release-please-version

.PHONY: publish
publish:
	KO_DOCKER_REPO=ghcr.io/robbert229/jaeger-postgresql/ ko resolve --base-import-paths -t $(VERSION) -f ./charts/jaeger-postgresql/values-template.yaml > ./charts/jaeger-postgresql/values.yaml
	helm package ./charts/jaeger-postgresql --app-version $(VERSION) --version $(VERSION) --destination=./hack/charts/
	helm push ./hack/charts/jaeger-postgresql-$(VERSION).tgz oci://ghcr.io/robbert229/jaeger-postgresql/charts

# plugin-start starts Jaeger-PostgreSQL
.PHONY: plugin-start
plugin-start:
	go run ./cmd/jaeger-postgresql --database.url=$(DBSTRING) --log-level=debug

# jaeger-start starts the all-in-one jaeger.
.PHONY: jaeger-start
jaeger-start:
	SPAN_STORAGE_TYPE='grpc-plugin' ./hack/jaeger-all-in-one --grpc-storage.server='127.0.0.1:12345' --query.enable-tracing=false

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

# tracegen-start starts a container that will produce test data spans. Useful for testing purposes.
.PHONY: tracegen-start
tracegen-start:
	docker run --rm --name jaeger-postgresql-tracegen --net=host \
		jaegertracing/jaeger-tracegen:1.55 -traces=1000

# tracegen-stop stops the jaeger-tracegen test data producer.
.PHONY: tracegen-stop
tracegen-stop:
	docker rm -f jaeger-postgresql-tracegen
