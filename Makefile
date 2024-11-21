# Uncomment to use OTEL Collector Builder installed by `install-ocb`
OCB=$(HOME)/bin/ocb

# Uncomment to use OTEL Collector Builder installed by
# `go install go.opentelemetry.io/collector/cmd/builder@v0.110.0`
#OCB=builder

all: go-test

## vendor: Vendors Go modules
vendor:
	go mod tidy && go work vendor
	cd observecol && go mod tidy && go work vendor
	cd components/processors/observek8sattributesprocessor && go mod tidy && go work vendor
	go mod tidy && go work vendor

## build: Build all Go packages
build:
	go build ./...

## test: Runs Go tests across all packages
go-test: build
	go list -f '{{.Dir}}' -m | xargs go test -v ./...
	
## release: Releases current tag through goreleaser
release:
	goreleaser release --clean

## install-ocb: Installs correct version of ocb binary
install-ocb:
	@mkdir -p "$(HOME)/bin"
	curl --proto '=https' --tlsv1.2 -L -o "$(HOME)/bin/ocb" https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/cmd%2Fbuilder%2Fv0.110.0/ocb_0.110.0_darwin_arm64
	@chmod +x "$(HOME)/bin/ocb"

## build-ocb: Builds project using ocb
build-ocb:
	$(OCB) --skip-compilation --config=builder-config.yaml
	sed -i -e 's/package main/package observecol/g' ocb-build/components.go
	sed -i -e 's/\/Users\/.*observe-agent\//..\//g' ocb-build/go.mod
	sed -i -e 's/\/home\/.*observe-agent\//..\//g' ocb-build/go.mod
	sed -i -e 's/observek8sattributesprocessor v0.0.0-00010101000000-000000000000 =>/observek8sattributesprocessor =>/g' ocb-build/go.mod
	cp ./ocb-build/components.go ./observecol/components.go
	cp ./ocb-build/go.mod ./observecol/go.mod
	cp ./ocb-build/go.sum ./observecol/go.sum
	go mod tidy && go work vendor
	cd ./observecol && go mod tidy && go work vendor

install-tools:
	cd ./internal/tools && go install go.opentelemetry.io/collector/cmd/mdatagen

.PHONY: all vendor build go-test release install-ocb build-ocb install-tools
