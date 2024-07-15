## go-test: Runs Go tests across all packages
go-test:
	go build ./...
	go test -v ./...

## release: Releases current tag through goreleaser
release:
	goreleaser release --clean

## install-ocb: Installs correct version of ocb binary
install-ocb:
	curl --proto '=https' --tlsv1.2 -L -o "$(HOME)/bin/ocb" https://github.com/open-telemetry/opentelemetry-collector/releases/download/cmd%2Fbuilder%2Fv0.104.0/ocb_0.104.0_darwin_arm64
	@chmod +x "$(HOME)/bin/ocb"

## build-ocb: Builds project using ocb
build-ocb:
	$(HOME)/bin/ocb --skip-compilation --config=builder-config.yaml
	sed -i -e 's/package main/package observeotel/g' observe-agent/components.go
	cp ./observe-agent/components.go ./cmd/collector/components.go
	go mod tidy
	go mod vendor 
