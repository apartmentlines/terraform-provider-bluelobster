# Repository Guidelines

## Project Structure & Module Organization
`main.go` is the provider entrypoint and embeds the version from `VERSION`. Core provider logic lives in `internal/provider/`, including resources, data sources, API client code, and `_test.go` files. Human-facing docs are under `docs/`, runnable Terraform examples are under `examples/`, and helper release scripts live in `scripts/`.

## Build, Test, and Development Commands
Use the `GNUmakefile` targets for routine work:

- `make fmt`: run `gofmt -w main.go internal/provider/*.go`.
- `make test`: run the full Go test suite with `go test ./...`.
- `make build`: compile the provider locally.
- `make install`: install the provider binary into your Go environment.
- `make tidy`: sync `go.mod` and `go.sum`.

For release prep, `scripts/build-all-platforms.sh` builds cross-platform artifacts and `scripts/generate-changelog-entries.sh` helps update `CHANGELOG.md`.

## Coding Style & Naming Conventions
This is a Go 1.23 project; keep code `gofmt`-clean and use tabs/default Go formatting. Follow existing naming patterns: exported provider types and constructors use `CamelCase`, internal helpers use `camelCase`, and Terraform-facing files are named by feature, such as `resource_instance.go` or `data_source_locations.go`. Keep provider-specific logic inside `internal/provider` rather than adding new top-level packages.

## Testing Guidelines
Tests use Go’s standard `testing` package and sit beside the code they cover in `internal/provider/*_test.go`. Name tests with clear behavior-focused prefixes, for example `TestClientCreateInstanceHostnodePayload`. Add or update tests whenever you change request shaping, schema behavior, state handling, or error translation.

## Security & Configuration Tips
Do not commit real API tokens or secret values. Prefer `BLUELOBSTER_API_TOKEN` for local testing, and use the write-only and ephemeral secret patterns already documented in `README.md` and `docs/`.
