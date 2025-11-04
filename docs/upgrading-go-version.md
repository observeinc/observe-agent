# Upgrading Go Version

This document describes the process for upgrading the Go version used by the observe-agent project.

## Prerequisites

1. Install the new Go version on your local machine
2. Verify the installation: `go version`
3. Ensure you're on a clean git branch

## Step-by-Step Process

### 1. Update Go Module Files

Update the `go` directive in all `go.mod` files to the new version:

```bash
# Main package
vim go.mod

# Observecol package
vim observecol/go.mod

# Component packages
vim components/processors/observek8sattributesprocessor/go.mod
vim components/receivers/heartbeatreceiver/go.mod
```

Change the line `go X.XX.X` to your new version (e.g., `go 1.24.8`).

### 2. Update README.md

Update the Go version reference in the README.md file:

```bash
vim README.md
```

Look for the section that mentions the Go version requirement and update it accordingly.

### 3. Update Dependencies

Run `go mod tidy` on the main package to update all dependencies:

```bash
go mod tidy
```

### 4. Update Vendored Dependencies

Run `make vendor` to vendor the updated dependencies:

```bash
make vendor
```

This command will:
- Run `go mod tidy && go work vendor` on the main package
- Run the same commands on the observecol package
- Run the same commands on component packages
- Update the final vendor directory

### 5. Update Go Workspace

Update the `go.work` file to reference the new Go version:

```bash
go work use
```

### 6. Update GitHub Workflow Files

Update all GitHub Actions workflow files that specify a Go version. Search for `go-version:` in all workflow files:

```bash
# Find all workflow files with go-version
grep -r "go-version" .github/workflows/

# Update each file
vim .github/workflows/tests-unit.yaml
vim .github/workflows/release-build.yaml
vim .github/workflows/tests-integration.yaml
vim .github/workflows/orca.yaml
vim .github/workflows/vuln-check-full.yaml
vim .github/workflows/vuln-check-release.yaml
vim .github/workflows/release.yaml
vim .github/workflows/release-nightly.yaml
```

Update any occurrences of:
- `go-version: X.XX.X` to your new version
- `go: [X.XX.X]` (in matrix strategies) to your new version

### 7. Run Tests on Component Packages

Run `go mod tidy` on each component package:

```bash
# observek8sattributesprocessor
cd components/processors/observek8sattributesprocessor
go mod tidy
cd ../../..

# heartbeatreceiver
cd components/receivers/heartbeatreceiver
go mod tidy
cd ../../..
```

### 8. Verify Builds

Test that all packages build successfully:

```bash
# Test main package build
go build

# Test component builds
cd components/processors/observek8sattributesprocessor
go build
cd ../../..

cd components/receivers/heartbeatreceiver
go build
cd ../../..
```

All builds should complete successfully. Minor compiler warnings from vendored dependencies are acceptable if they don't affect the build.

### 9. Run Tests

Run the test suite to ensure everything still works:

```bash
make go-test
```

### 10. Commit Changes

Once all tests pass, commit your changes:

```bash
git add .
git commit -m "chore: upgrade to Go X.XX.X"
```

## Files That Need to Be Updated

Here's a comprehensive checklist of files that need to be updated:

### Go Module Files (4 files)
- [ ] `go.mod`
- [ ] `observecol/go.mod`
- [ ] `components/processors/observek8sattributesprocessor/go.mod`
- [ ] `components/receivers/heartbeatreceiver/go.mod`

### Documentation (1 file)
- [ ] `README.md`

### GitHub Workflow Files (8 files)
- [ ] `.github/workflows/tests-unit.yaml`
- [ ] `.github/workflows/release-build.yaml`
- [ ] `.github/workflows/tests-integration.yaml`
- [ ] `.github/workflows/orca.yaml`
- [ ] `.github/workflows/vuln-check-full.yaml`
- [ ] `.github/workflows/vuln-check-release.yaml`
- [ ] `.github/workflows/release.yaml`
- [ ] `.github/workflows/release-nightly.yaml`

### Automatically Updated Files
These files will be updated automatically by running the commands above:
- `go.work` (updated by `go work use`)
- `go.sum` (updated by `go mod tidy`)
- `vendor/modules.txt` (updated by `make vendor`)
- All `go.sum` files in component packages

## Troubleshooting

### Build fails with "go.work lists go X.XX.X"

If you see an error like:
```
go: module . listed in go.work file requires go >= X.XX.X, but go.work lists go Y.YY.Y
```

Run: `go work use` to update the workspace file.

### Dependency conflicts

If you encounter dependency conflicts after running `go mod tidy`, try:

1. Clear the module cache: `go clean -modcache`
2. Re-run `go mod tidy`
3. Re-run `make vendor`

### Component builds fail

If component builds fail, make sure to:
1. Run `go mod tidy` in each component directory
2. Ensure the component's `go.mod` has been updated to the new Go version

## Additional Notes

- The Go version specified in `go.mod` is the minimum required version
- When upgrading Go, review the [Go release notes](https://go.dev/doc/devel/release) for any breaking changes
- Test thoroughly before merging, especially if there are major version changes
- Consider running integration tests after the upgrade
