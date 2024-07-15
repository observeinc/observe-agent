## go-test: Runs Go tests across all packages
go-test:
	go build ./...
	go test -v ./...

## release: Releases current tag through goreleaser
release:
	goreleaser release --clean

## build-ocb: Builds observeotel binary
build-ocb:
	$(HOME)/bin/ocb --skip-compilation --config=builder-config.yaml
	sed -i -e 's/package main/package observeotel/g' observe-agent/components.go
	cp ./observe-agent/components.go ./cmd/collector/components.go
	modmerge go.mod observe-agent/go.mod
	go mod tidy
	go mod vendor 
