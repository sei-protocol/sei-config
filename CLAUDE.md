# sei-config

Shared Go library providing unified configuration types, mode-aware defaults, validation, and serialization for all Sei node components (`seid`, `seictl`, `sei-k8s-controller`).

## Architecture

- **Package**: `seiconfig` (import as `github.com/sei-protocol/sei-config`)
- **Unified struct**: `SeiConfig` — single type covering all fields from both `config.toml` (Tendermint) and `app.toml` (Cosmos SDK + Sei)
- **Legacy IO**: Reads/writes the existing two-file layout via intermediate legacy types in `legacy.go`; atomic writes (temp + rename) prevent corruption
- **No external dependencies** beyond `github.com/BurntSushi/toml`

### Key Files

| File | Purpose |
|------|---------|
| `config.go` | `SeiConfig` and all sub-config struct definitions |
| `types.go` | `NodeMode`, `Duration`, `WriteMode`, `ReadMode` |
| `defaults.go` | `DefaultForMode()` — mode-aware baseline configs |
| `validate.go` | `Validate()` — structured diagnostics (Error/Warning/Info) |
| `resolve.go` | `ResolveEnv()` — `SEI_`/`SEID_` env var resolution via reflection |
| `io.go` | `ReadConfigFromDir()`, `WriteConfigToDir()`, `ApplyOverrides()` |
| `legacy.go` | Two-file TOML mapping types and `SeiConfig` ↔ legacy conversion |

## Code Standards

### Go

- **Idiomatic Go above all.** Prefer clarity over cleverness. Three explicit lines beat a cryptic one-liner.
- **No unnecessary abstractions.** This is a leaf library — keep it flat. Don't add interfaces until there are two concrete implementations.
- **Zero `panic`.** Every failure path must return an error. Callers (seid, seictl, controller) decide how to handle it.
- **All code must pass `golangci-lint`** (config in `.golangci.yml`). Fix lint issues at the source — do not add `nolint` directives without a comment explaining why suppression is the only option.
- **Imports grouped**: stdlib, external, then `github.com/sei-protocol/sei-config` (enforced by goimports).
- **Exported types are the contract.** Every exported type, function, and constant must have a doc comment. Unexported helpers don't need them unless the logic is non-obvious.
- **Struct tags are the schema.** TOML tags on `SeiConfig` define the unified `sei.toml` key names. Legacy TOML tags on `legacy*.go` types must exactly match the existing `config.toml`/`app.toml` key names — do not change them without a migration plan.
- **Keep `SeiConfig` and legacy types in sync.** Every field added to `SeiConfig` must have corresponding legacy mapping in `toLegacyTendermint()`/`toLegacyApp()` and `fromLegacy()`. Tests enforce round-trip fidelity.

### Defaults & Modes

- `DefaultForMode(mode)` is the single entry point. Baseline defaults live in `baseDefaults()`, mode overrides in `apply*Overrides()`.
- When adding a new field: add to `baseDefaults()` with the safe/common default, then add mode-specific values only where they differ.
- Every mode's defaults must pass `Validate()` — this is enforced by `TestDefaultForMode_AllModesValid`.

### Validation

- Validation returns `*ValidationResult` with typed `Diagnostic` entries — never `error` directly.
- `SeverityError` = blocks startup. `SeverityWarning` = logged. Use the right severity.
- Cross-field checks go in `validateCrossField()`.

### Testing

- Tests use the `testing` package only — no assertion libraries, no test frameworks.
- Run tests with `make test` before submitting changes.
- Every new field should be exercised in at least one round-trip test (`WriteConfigToDir` → `ReadConfigFromDir`).
- Test file naming: `config_test.go` for the main test file. Add `*_test.go` files when a single file grows past ~500 lines.

## Build & Validate

```bash
make test         # Run all tests
make lint         # Run golangci-lint
make vet          # Run go vet
make fmt          # Format all code (gofmt -s)
make ci           # lint + vet + test (what CI runs)
make test-cover   # Tests with coverage report
```

## Design Constraints

- **This is a library, not a binary.** There is no `main` package. Consumers are `seid`, `seictl`, and `sei-k8s-controller`.
- **Minimal dependencies.** Think hard before adding a new `require`. Every dependency here transitively affects three other repos.
- **Phase-aware IO.** `ReadConfigFromDir`/`WriteConfigToDir` currently handle the two-file layout (Phase 2). When Phase 3 ships unified `sei.toml`, the IO layer switches internally — callers should not need to change.
- **Backward compatibility matters.** The legacy types in `legacy.go` must produce TOML files that existing `seid` binaries can read. Changing a legacy TOML tag is a breaking change.
