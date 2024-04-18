## go-test: Runs Go tests across all packages
go-test:
	go build ./...
	go test -v ./...

## release: Releases current tag through goreleaser
release:
	goreleaser release --clean