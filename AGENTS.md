# dupfind

Duplicate code finder.

## Code Map

- `cmd/dupfind/main.go`: executable entrypoint and duplicate detection logic.
- `cmd/dupfind/main_test.go`: unit tests.
- `cmd/dupfind/testdata`: test fixtures.

## Style

- Write boring, idiomatic Go.
- Keep packages small and purpose-named.
- Prefer simple flat code over clever abstraction.
- Put interfaces at consumer boundaries, not in front of every type.
- Constructor-inject IO, clocks, environment, clients, and collaborators.
- Name meaningful literals as constants.
- Return explicit errors; fail fast; do not hide failures.
- Add third party dependencies only with user approval.

## Tests

- Use `make test` for quick iteration (fast tests only); use `make test-slow` or `make pre-commit` for the full suite.
- Slow tests use `//go:build slow` and run with `-tags=slow`; `make pre-commit` always includes them.
- Unit tests first; integration tests second; end-to-end tests last.
- Tests use problem-domain language.
- Use table tests when they make cases clearer.
- Use fully functional fakes, not mocks.
- No skipped tests or soft-failing assertions.
- Boundary rules belong in tests when practical.
