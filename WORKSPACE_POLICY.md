# Workspace Policy

## Package manager
- JavaScript/TypeScript: `pnpm@10.6.2` (root `packageManager` is pinned).
- Go: native Go modules with root `go.work`.
- Flutter/Dart: `flutter` and `dart` from pinned toolchain.
- Rust: stable pinned in `.tool-versions`.

## Version pinning
- Go: `go 1.24` in each `go.mod`, coordinated by `go.work`.
- Node: `>=22.12.0 <23`.
- pnpm: exact `10.6.2`.
- Flutter: `3.24.0` target.
- Rust: `1.82.0` target.

## Formatting and lint strategy
- Go: `gofmt`, `go vet`, `staticcheck` (TODO in CI phase 2).
- TS/JS: `eslint` + `prettier` (TODO config rollout).
- Dart: `dart format` + `flutter analyze`.
- Rust: `cargo fmt` + `cargo clippy`.

## CI strategy
- PR checks:
  - Go compile/test across all Go modules.
  - Node install + build checks for `admin-web` and `manager-desktop`.
  - Flutter analyze for mobile client (partial scaffold warning allowed).
  - OpenAPI lint (TODO: spectral).
- Main branch:
  - Build docker images.
  - Generate SBOM artifacts.
  - Sign artifacts (TODO: cosign integration).
