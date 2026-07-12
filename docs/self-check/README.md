# Self-Check Integration

Run `dupfind` against its own codebase to prevent copy-paste from accumulating
in the tool itself, and wire the check into the `pre-commit` pipeline.

## Current Behaviour

The `Makefile` has a `dup-check` target that is explicitly disabled:

```makefile
dup-check:
	@echo "dup-check: disabled"
```

The pre-commit hook runs `tidy-fmt-check`, `vet`, `cyclo`, `coverage`, and
`gtags`, but never invokes `dupfind` on its own source tree.

## Motivation

A duplicate-code detector that does not detect duplicates in its own codebase
is incongruous. Activating self-check serves three purposes:

1. **Dogfooding.** If `dupfind` produces false positives or misses real
   duplicates in its own small codebase, the authors discover and fix those
   issues immediately.
2. **Regression prevention.** As the tool grows, duplicated error-handling
   stanzas, flag-parsing boilerplate, or test helpers may get copy-pasted.
   Self-check catches this before the pattern spreads to three or four copies.
3. **Credibility.** A reviewer who sees `dup-check: disabled` in the Makefile
   may reasonably wonder whether the tool works at all on real code.

## Design

### Expected baseline

At the current size (~440 lines, one file, one package), `dupfind` on itself
should produce:

```
scanned 2 files, extracted N blocks (min 30 nodes, 5 lines)

=== Exact Duplicates (0 groups) ===

=== Near Duplicates (0 pairs, similarity >= 0.75) ===
```

The exact counts depend on whether `main_test.go` is included (`-tests`) and
what the minimum thresholds surface. The sibling-case patterns in `caseVisitor`
and the duplicated `addBlock` call pattern in the `blockVisitor.Visit` switch
are near-duplicate candidates worth monitoring.

### Threshold tuning

Self-check may require relaxed thresholds (`-min-nodes=15`, `-min-lines=2`,
`-sim=0.70`) to detect evolving duplication before it becomes entrenched.
Store these in the Makefile rather than hardcoding them in `main.go`.

### Makefile integration

```makefile
dup-check: build
	@echo "dup-check: scanning dupfind source tree"
	@./bin/dupfind -root . -min-nodes 15 -min-lines 2 -sim 0.70 -min-copies 2
	@echo "dup-check: ok"
```

Wire into `pre-commit`:

```makefile
pre-commit: tidy-fmt-check vet cyclo coverage gtags dup-check
	@echo "pre-commit: ok"
```

### Testdata exclusion

Self-check must exclude `testdata/` directories, which contain deliberately
duplicate code. The existing `collectGoFiles` already skips `.git`, `vendor`,
`.gocache`, and `node_modules`. Add `testdata` to the skip list, or pass a
`-skip-dirs` flag.

Alternatively, run the check from `..` — scanning internal/dupfind/ only and
not testdata. This is simpler and avoids adding a skip mechanism.

## Risk

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Self-check fails on known test-data duplicates | High (if testdata scanned) | Low | Exclude testdata from scan scope |
| Self-check fails on `caseVisitor` sibling patterns | Medium | Low | Relax threshold or document expected false positives |
| Self-check takes noticeable time | Low (tiny codebase) | Low | Not a concern at current size |

## Acceptance

- `make dup-check` runs `dupfind` on the project source (excluding testdata).
- `make pre-commit` includes `dup-check`.
- A deliberate copy-paste into the source tree causes `make pre-commit` to
  fail.
- Running `make dup-check` on the current tree produces zero reported
  duplicates (exact or near) at the chosen thresholds.
