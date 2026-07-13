## Analysis: `dupfind` — Structural Duplicate Code Finder

### Domain

`dupfind` detects exact and near-duplicate code blocks in Go source trees. It operates in two phases: an exact-match pass using AST tree signatures (structural serialization), and a near-duplicate pass using IDF-weighted Jaccard similarity over k-shingled token streams. The audience is a Go engineer maintaining a codebase who wants to surface copy-paste before it ossifies into maintenance debt.

### Architecture

The entire program is one file (`main.go`, ~440 lines). It has no third-party dependencies and uses only the standard library (`go/ast`, `go/parser`, `go/token`, `flag`). This is consistent with the project's stated preference for zero dependencies without user approval.

The pipeline is linear:

1. **File collection** — walks a directory tree, excludes `vendor`, `.git`, `node_modules`, `_test.go` (opt-in via `-tests`).
2. **Block extraction** — walks AST per file with `ast.Walk`, collects function bodies, if/for/range/switch/select bodies, and case/comm clauses. Each block below the minimum node count (default 30) or line span (default 5) is discarded.
3. **Exact duplicate detection** — groups blocks by signature; groups with `>= min-copies` (default 3) are reported.
4. **Near-duplicate detection** — computes IDF over shingles across all blocks, builds an inverted index for candidate-pair generation, filters exact dups, nested blocks, and sibling cases, then evaluates weighted Jaccard against a threshold (default 0.75).

### Core Design Decisions

**Identifier-preserving signatures.** The tree signature preserves identifier names as `id:name`, literal kinds as `lit:KIND`, operators as `bin+`, `un-`, etc. This is the correct choice for exact-match detection: a copy-paste preserves identifiers; a refactor might not. The test `TestSignatureDoesNotNormalizeAllIdentsToID` verifies that unrelated functions with the same structure (read-file, error-wrap, iterate, count) do *not* collide. This is the single most important design decision — it controls the precision/recall tradeoff.

**IDF-weighted Jaccard for near-duplicates.** This is the right approach. Go has pervasive boilerplate (`if err != nil`), and unweighted Jaccard would inflate similarity for any two blocks that both read a file and check an error. Computing IDF across the corpus and weighting rare shingles higher means that a `len(line) > 0` ↔ `len(line) >= 1` change has structural weight, while `if err != nil` has near-zero weight.

**Sibling-case filtering.** The `siblings()` function suppresses case clauses that share the same parent switch/select range. This is necessary because switch cases are structurally similar by design — every case in a dispatch function has the same shape `fmt.Sprintf("...", cmd)`. Without this filter, every switch statement would be a false positive storm.

**Inverted index for candidate generation.** Rather than computing O(n²) pairwise comparisons (which would be prohibitive for large codebases), near-duplicate detection builds an inverted index mapping each shingle to the block indices that contain it. Candidates are generated only from blocks that share at least one shingle. This is a standard and correct optimization.

### Verification Surface

The test suite (`main_test.go`, ~280 lines) covers:

| Surface | Coverage |
|---|---|
| Block extraction thresholds | Tests `minNodes` + `minLines` filtering across exact, falsepos dirs |
| Exact duplicate grouping | Detects 2 copies in `exact/`, 3 copies in `mincopies/`, 0 in `falsepos/` |
| Near-duplicate detection | Detects 1 pair in `near/`, 0 in `falsepos/` with threshold 0.75 |
| Signature identifier preservation | Verifies `id:data`, `id:count`, `id:ReadFile` in signatures |
| Signature literal-kind distinction | Verifies `lit:STRING` vs `lit:INT` are differentiated |
| IDF weighting | Verifies boilerplate shingles (every block) have lower weight than rare shingles (one block) |
| Weighted Jaccard suppression | Verifies falsepos pair scores below 0.75 |
| Shingling correctness | Boundary cases: k > len(tokens), k == len(tokens), empty token list, repeated tokens |
| File collection | Excludes `_test.go` without `-tests`, includes with |
| Function naming | Plain functions, value receivers, pointer receivers |
| Sibling filtering | Verifies case clauses in same switch are reported as siblings, not as near-duplicates |
| Min-copies threshold | Verifies 2 copies filtered by min-copies=3, 3 copies detected by min-copies=3 |
| Line-span filtering | Verifies `minLines=100` excludes all blocks |

### Observations

**Single file is the right call at this size.** At ~440 lines, the concerns are mixed (file walking, AST walking, signature computation, shingling, IDF computation, output formatting) but none of them are independently complex enough to justify a package boundary. The cost of splitting — more files to open, more import paths to trace, more public surfaces to reason about — exceeds the benefit until any single concern crosses ~200 lines of its own. The current code is readable from top to bottom in one pass. If extraction thresholds, signature computation, or the near-duplicate pipeline grow substantially, the natural fracture lines are self-evident from the section comments.

**The `blockVisitor` / `caseVisitor` pair is the most cognitively demanding construct.** The `caseVisitor` wraps `blockVisitor` and passes parent range into case clauses, then restores the original visitor for nested block extraction. This works but requires the reader to hold two visitor types and their switching logic in mind. A simpler alternative: extract blocks in a two-pass approach — first identify all blocks, then for case clauses, walk up to find the parent switch range.

**`siblings()` uses `parentStart/End`, but this field is only set inside switch/select bodies.** The default is `0,0`, so `siblings()` correctly returns false for non-case blocks. But if a user introduces a new block type that happens to set `ParentStart/ParentEnd` for another reason, `siblings()` would produce false negatives. This is a latent fragility, not a current bug.

**No fast-path for identical signatures in near-duplicate detection.** The `findNearDuplicates` function explicitly skips pairs where `blocks[a].Sig == blocks[b].Sig` ("exact duplicate, already reported"), but this check is O(1) key comparison. The inverted index still generates candidate pairs for exact duplicates, only to discard them. This is a minor performance issue — exact-duplicate blocks could be excluded from the near-dup pass entirely.

**The `near/filler.go` file is an interesting design choice.** It injects a third function that shares Go boilerplate with the two near-duplicate variants. This creates realistic IDF pressure: `for _, item := range ...` and `return ..., fmt.Errorf(...)` appear in `unrelated`, so their shingles get lower weight. The test `TestIDFWeightsBoilerplateLow` validates this. This is good test data discipline.

### What is Missing

- **No nested-block extraction depth control.** The code already extracts nested blocks independently (if inside for inside func yields three blocks). The question is whether deeper extraction or overlapping-block dedup adds value. Decision: let it stay missing. Deep nesting is already caught by cyclomatic complexity checking, and no motivating example case has been found that survives default thresholds (minNodes=30, minLines=5, minCopies=3). Revisit if a real case emerges.

- **No aggregation across near-duplicate pairs.** `findNearDuplicates` returns pairwise results but does not cluster them into groups. A block with three near-copies produces three pairs. The output presents each pair independently, which can be noisy at scale.

- **No Makefile target for the tool on itself.** The `dup-check` target in `Makefile` is explicitly disabled: `"dup-check: disabled"`. The tool is not applied to its own codebase. Running `dupfind` on `dupfind` would be a zero-exact, zero-near result (one file, one block, no pairs), so this is not hiding anything interesting, but the habit of self-verification is valuable.

### Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| False positive from shared Go boilerplate | Medium | Low | IDF weighting + `siblings()` filter handle the common cases; `falsepos/` testdata validates |
| Performance on large codebases | Medium | Medium | Inverted index keeps candidate generation near-linear in practice; no minhash or LSH fallback for >10k blocks |
| Case clause false negatives from `siblings()` change | Low | Low | No current code sets `ParentStart/End` outside switch/select |
| New maintainer confusion around `caseVisitor` | Medium | Low | ~50 lines of well-named code inline; the pattern is singletons with one reference each |

### Conclusion

This is a well-scoped, dependency-free tool that makes a clear design choice (identifier-preserving signatures, IDF-weighted Jaccard) and validates it thoroughly with test data designed to exercise the precision-recall boundary. The implementation is boring and idiomatic Go. The most fragile area is the `blockVisitor`/`caseVisitor` interaction, which is contained and tested. The test data documents the tool's behavior more clearly than any prose could — each directory (`exact/`, `near/`, `falsepos/`, `siblings/`, `mincopies/`) is a single-inductance assertion about what dupfind should and should not flag.

The tool earns its keep.
