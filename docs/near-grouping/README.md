# Near-Duplicate Group Aggregation

Cluster near-duplicate blocks into groups (not pairwise) so that a block with
multiple near-copies produces one report entry instead of N-1 redundant pairs.

## Current Behaviour

`findNearDuplicates` returns a flat list of `nearPair` structs, sorted by
descending similarity. A block that has three near-copies (A→B, A→C, B→C)
produces three independent pairs, each listed with its own similarity score.
The reader must mentally merge these to understand that blocks A, B, and C
form one cluster.

For a 10-copy cluster this produces 45 pairs. The output is noisy enough that
the signal (four distinct clusters) is hidden inside the noise (45 nearly
identical rows).

## Motivation

In practice, near-duplicate code appears in clusters. When a developer copy-
pastes a block three times across a codebase, the interesting output is:

```
[near-dup group 1] 3 copies
    file_a.go:10-25   sim 0.92
    file_b.go:30-45   sim 0.88
    file_c.go:5-20    reference
```

The pairwise form requires the reader to assemble this from three rows,
inferring transitivity from similarity values.

## Design

### Cluster algorithm

1. Build a graph where each block is a vertex and each near-duplicate pair
   (sim >= threshold) is an edge.
2. Compute connected components.
3. For each component with >= 2 vertices, report one group.

Within a group, designate the block with the most edges (highest connectivity)
as the "reference" and report each other block's similarity to it. If multiple
blocks have the same connectivity, use the one with the highest average
similarity to the rest.

### Output format

```
=== Near Duplicate Groups (3 groups) ===

[1] 3 copies, 24-28 nodes:
    reference: file_a.go:10-25  processData (28 nodes)
    variant:   file_b.go:30-45  processData (28 nodes)  sim 0.92
    variant:   file_c.go:5-20   handleItem (24 nodes)   sim 0.88

[2] 2 copies, 12-15 nodes:
    reference: file_d.go:50-60  parseLine (15 nodes)
    variant:   file_e.go:80-90  parseLine (15 nodes)    sim 0.81
```

### Retaining pairwise output

Add a `-pairs` flag that outputs the current flat pairwise form. This is
useful for script consumption. The default is grouped output.

## Sibling and Container Filter Interaction

Grouping must respect the existing filters:

- Pairs filtered by `siblings()` or `contains()` are not added as edges.
- If filtering leaves a component with only one vertex, it is not reported.

## Test Data

Add a `testdata/near-group/` directory containing:

- Three files, each with a near-copy of the same block (A↔B sim 0.85, A↔C
  sim 0.80, B↔C sim 0.78).
- Expected: one group of 3 copies, not three pairs.

## Acceptance

- Grouped output is default; `-pairs` restores flat pairwise form.
- Components correctly handle transitive similarity (A~B, B~C → group A,B,C
  even if A~C falls below threshold).
- Single-vertex components are not reported.
- Existing `-sim` threshold, `siblings()`, and `contains()` filters still apply.
- New functionality is done primarily by new code. Avoid rewriting code unnecessarily.