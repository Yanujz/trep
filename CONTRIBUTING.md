# Contributing

## Development setup

```sh
git clone https://github.com/Yanujz/trep
cd trep
go build ./...   # verify it compiles
go test ./...    # run the test suite
```

Requires **Go 1.21+**. No other tools are needed; dependencies are vendored.

## Running tests

```sh
make test          # run all tests
make test-race     # run with race detector (recommended before pushing)
make test-cover    # generate a coverage report
```

## Adding a new test input format

1. Create `pkg/parser/<name>/parser.go` implementing `parser.Parser`.
2. Register it via `func init() { parser.Register(Parser{}) }`.
3. Add a blank import in `cmd/trep/main.go`.
4. Write tests in `pkg/parser/<name>/parser_test.go`.

Adding a new coverage format follows the same pattern under `pkg/coverage/parser/`.

## Pull request guidelines

- Keep changes focused; one concern per PR.
- All tests must pass (`make test-race`).
- Update the relevant flags table in `README.md` if you add or change CLI flags.
- Do not add new runtime dependencies without discussion.

## Reporting bugs

Open a GitHub issue with:
- The `trep` version (`trep --version`).
- The input file or a minimal reproducer.
- The observed and expected behaviour.
