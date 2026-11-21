<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/assets/banner-hero-dark.svg">
  <img src="docs/assets/banner-hero-light.svg" alt="go-lint: one lint kit for every Versatile Go repository. The same rules, the same config, checked the same way everywhere." width="100%">
</picture>

One lint kit for Go repositories that follow the Versatile engineering standard: a generated golangci-lint config, a drift check for it, and seven analyzers for what stock linters cannot see.

[Quick start](#quick-start) · [Rules](#usage) · [Layout](#the-folder-layout) · [Configuration](#configuration) · [Development](#development-and-tests) · [License](#license)

![license](https://img.shields.io/badge/license-GPL--3.0--or--later-c6f432) ![go](https://img.shields.io/badge/go-1.27-c6f432)

## What it is

go-lint is a versioned Go module with one command, `vlint`. A repository runs it with `go run` at a pinned version, so nothing is added to its `go.mod` and every machine runs the same rules.

| It does | It does not |
| --- | --- |
| Write the repository's `.golangci.yml` from one shared config plus a small overlay | Replace golangci-lint: it generates its config and runs beside it |
| Fail the check when the committed config drifts from the generated one | Let a repository enable extra linters or loosen shared settings |
| Run seven `go/analysis` analyzers (layout, clock, environment, errors, tests, comments, design-record references) | Apply risky autofixes: only `errorlint` and the formatters |

## How it works

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/assets/diagram-flow-dark.svg">
  <img src="docs/assets/diagram-flow-light.svg" alt="How go-lint runs. Your repository's check command calls vlint through go run at a pinned version. vlint golangci writes .golangci.yml from the shared config plus your overlay, and vlint golangci -check fails when the committed file drifts. vlint ./... runs the seven analyzers: layout, clockseam, envroot, errstring, noskipenv, doccomment and adrref. golangci-lint then runs with the generated config." width="100%">
</picture>

1. The repository's `make check` calls `vlint` with `go run github.com/versatile-sh/go-lint/cmd/vlint@<version>`.
2. `vlint golangci -check` regenerates the config in memory and fails when `.golangci.yml` differs.
3. `vlint ./...` runs the analyzers over every package, test files included where a rule covers them.
4. `golangci-lint run ./...` runs the stock linters with the generated config.

## Quick start

From the root of a Go module, with Go 1.27 and golangci-lint 2.15:

```sh
go run github.com/versatile-sh/go-lint/cmd/vlint@v0.1.0 golangci
go run github.com/versatile-sh/go-lint/cmd/vlint@v0.1.0 ./...
golangci-lint run ./...
```

The first line writes `.golangci.yml`. The second prints one line per finding as `file:line:col: message` and exits non-zero when there is any. Wire both into the check command:

```make
VLINT := go run github.com/versatile-sh/go-lint/cmd/vlint@v0.1.0

check:
	$(VLINT) golangci -check
	$(VLINT) ./...
	golangci-lint run ./...
```

## Usage

| Analyzer | Reports | Fix |
| --- | --- | --- |
| `layout` | A folder outside the layout below, or an import that points against the layers | Move the package; invert the dependency through a port |
| `clockseam` | `time.Now`, `time.Since`, `time.Until` (calls or function values) outside the composition root | Inject a clock port; the real clock lives in `app/` or a `clock*` package |
| `envroot` | `os.Getenv`, `os.LookupEnv`, `os.Environ` outside the composition root | Read configuration once in `app/` and pass values in |
| `errstring` | `err.Error()` compared with `==`, `!=`, `switch` or a `strings` matcher, outside tests | A sentinel with `errors.Is`, or a typed error with `errors.As` |
| `noskipenv` | A test that skips silently: any `Skip` not guarded by `testing.Short()`, an opt-out variable or a platform check | `t.Fatal` when the environment is missing; skip only under `testing.Short()`, an opt-out variable whose name has a `SKIP` token (`STORE_SKIP_INTEGRATION=1`), or `runtime.GOOS`/`GOARCH` |
| `doccomment` | An exported symbol without a doc comment, a doc comment over two lines, a doc comment on an unexported symbol, and any other comment | Name things so the code reads without comments; keep two lines on exported symbols |
| `adrref` | A design-record id (`ADR-12`) or a record folder path (`docs/adr/`) in a comment or string literal | State the behaviour; keep records outside the repository |

Exempt from `clockseam` and `envroot`: `main` packages, `app/` and `cmd/`, `_test.go` files, and test-kit packages whose name ends in `test`. `doccomment` allows directives (`//go:`, `//nolint:`, build tags), generated files, comments above the `package` clause, and two-line doc comments on struct fields and interface methods. `noskipenv` exempts the re-exec idiom, a test function named `*HelperProcess`.

Run one analyzer with its flag, for example `vlint -layout ./...`. `vlint help` lists them. Files behind a build tag are analyzed only when the tag is set: `GOFLAGS=-tags=integration vlint ./...`.

Apply the safe fixes only:

```sh
go run github.com/versatile-sh/go-lint/cmd/vlint@v0.1.0 fix ./...
```

It runs `golangci-lint run --fix --enable-only errorlint` and then `golangci-lint fmt` (gofmt and goimports). Turning `%v` into `%w` makes wrapped errors visible to `errors.Is`, so run the tests afterwards.

### The folder layout

Paths are relative to the Go module root.

| Folder | Role | May import from the module |
| --- | --- | --- |
| `app/` | The composition root: configuration, wiring, the real clock | Anything. Only `cmd/` and `main` packages import it |
| `cmd/<binary>/` | Thin mains and generators | Anything. Nothing imports it |
| `port/` | The contract: interfaces, the plain types they speak, sentinel errors, the coded error type | `port/`, `schema/`, `api/` |
| `core/` | Pure domain rules, no I/O | `core/`, `port/`, `schema/` |
| `service/` | Operations over core and ports; knows no transport | Anything but `app/` and `cmd/` |
| `schema/`, `api/` | Contract data and generated types | Their own subtree (`api/` may use `schema/`) |
| Flat adapter folders (`store/`, `server/`, one per driver) and `internal/<helper>/` | Adapters and module-private helpers | Anything but `app/` and `cmd/` |

Test files may import anything except `cmd/`. These folder names are reported wherever they appear at the root or under `internal/`: `domain`, `ports`, `application`, `adapters`, `adapter`, `bootstrap`, `infrastructure`, `infra`, `usecase`, `usecases`, `pkg`, `util`, `utils`, `common`, `helpers`, `shared`, `lib`. Layer folders sit at the module root, not under `internal/`. Finer edges between adapters belong to the repository's own dependency table.

## Configuration

A repository adds rules in `lint/golangci.local.yml`, then runs `vlint golangci` to regenerate. The overlay accepts two keys only:

```yaml
exclusions:
  rules:
    - path: 'legacy/'
      linters: [gosec]
depguard:
  rules:
    core-no-sql:
      files: ["**/core/**"]
      deny:
        - pkg: database/sql
          desc: core does no I/O
```

Any other key is refused, and a depguard rule may not reuse a shared rule's name. The shared config leaves `sloglint` autofix unused, and `nilnil` checks pointers, functions and channels only, so `(nil, nil)` from gRPC metadata stubs and `any` results is not reported.

## Development and tests

```sh
make check   # gofmt, go vet, config drift, vlint on itself, golangci-lint
make test    # go test -race -count=1 ./...
make images  # regenerate the README images in docs/assets
```

Each analyzer has `analysistest` cases in `passes/<analyzer>/testdata`: every rule has a case that must be reported and a case that must not. The README images are generated by `docs/assets/src/build-banners.mjs`; never edit an SVG by hand.

## Security notes

go-lint reads source files and `go.mod` files and writes only `.golangci.yml`. `vlint fix` runs the `golangci-lint` binary found on `PATH`. Report a vulnerability through a private GitHub security advisory on this repository, not a public issue.

## Contributing

Open an issue or a pull request. A new rule comes with its analyzer, its `analysistest` cases and a row in the table above, and `make check` and `make test` pass.

## License

Copyright (C) 2026 Versatile (versatile.sh).

go-lint is free software: you can redistribute it and modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version. It is distributed without any warranty; see [`LICENSE`](LICENSE) for the full terms.

A commercial license, for use under terms other than the GPL, is available from Versatile at <https://versatile.sh>.
