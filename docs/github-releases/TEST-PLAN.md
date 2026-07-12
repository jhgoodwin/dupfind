# Release Test Plan

Prove the two-track release system works end-to-end: a first release from scratch, then an incremental release driven by a real feature.

---

## Phase 1 — First release

### Prerequisites

- Remote GitHub repo has **no existing tags** (v0.*.*, unstable, latest, or otherwise).
- `master` branch exists with the release workflows in `.github/workflows/`.
- `go.mod` module path is set to `github.com/jhgoodwin/dupfind` (required for `go install`).

### Steps

**1. Trigger the unstable build**

```bash
git push origin master
```

**2. Verify the unstable pre-release**

Go to **Actions** → **Release (Unstable)**.

- [ ] Workflow completed successfully.
- [ ] Pre-release created with tag `v0.1.0-unstable.YYYYMMDD.N` (e.g. `v0.1.0-unstable.20260712.1`).
- [ ] Tag `unstable` exists in the repo and points to this commit.
- [ ] Build artifact is attached (the binary from `make build`).

**3. Promote to stable**

Go to **Actions** → **Promote to Stable** → **Run workflow**.
- Bump type: `release` (no version bump — promotes the base as-is).

**4. Verify the stable release**

- [ ] GitHub Release `v0.1.0` appears with a "Release" badge (not "Pre-release").
- [ ] Tag `v0.1.0` exists in the repo.
- [ ] Tag `latest` exists and points to the same commit as `v0.1.0`.
- [ ] Tag `v0.1.0-unstable.YYYYMMDD.N` still exists (immutable).
- [ ] Tag `unstable` still exists (unchanged — no new push happened).

**5. Verify `go install`**

```bash
go install github.com/jhgoodwin/dupfind@latest
```

- [ ] Installs successfully.
- [ ] Binary runs (`dupfind -h`).

---

## Phase 2 — Incremental release via self-check feature

Implement the feature described in [`docs/self-check/README.md`](../self-check/README.md):

- Enable `dup-check` in the Makefile.
- Wire it into `pre-commit`.
- Exclude `testdata/` from the scan scope so self-check doesn't flag deliberate test fixtures.

### Steps

**1. Implement and push the feature**

```bash
# Make the changes described in docs/self-check/README.md
git add -A
git commit -m "feat: enable self-check"
git push origin master
```

**2. Verify the first unstable build after the feature**

Go to **Actions** → **Release (Unstable)**.

- [ ] Workflow completed successfully.
- [ ] Pre-release tag: `v0.1.0-unstable.YYYYMMDD.N+1` (build counter incremented from phase 1).
- [ ] `unstable` tag moved to this new commit.

**3. Make a trivial second push to prove per-day build counting**

```bash
# Any trivial change, e.g. fix a typo in README
git add -A
git commit -m "docs: fix typo"
git push origin master
```

- [ ] Second unstable build created: `v0.1.0-unstable.YYYYMMDD.N+2`.
- [ ] `unstable` tag moved again.

**4. Promote to stable**

Go to **Actions** → **Promote to Stable** → **Run workflow**.
- Bump type: `patch`.

**5. Verify the incremental release**

- [ ] GitHub Release `v0.1.1` appears.
- [ ] Tag `v0.1.1` exists.
- [ ] Tag `latest` now points to `v0.1.1`.
- [ ] Tag `v0.1.0` still exists and points to its original commit (immutable).
- [ ] The `v0.1.0` release is untouched.

**6. Verify `go install` resolves to the new stable**

```bash
go install github.com/jhgoodwin/dupfind@latest
```

- [ ] Installs `v0.1.1`, not `v0.1.0`.

**7. Verify self-check works**

```bash
make dup-check
```

- [ ] Runs `dupfind` on its own source tree with no false positives.
- [ ] Exits zero (no duplicates found at the chosen thresholds).

---

## Test matrix summary

| What's proved | How |
|---|---|
| Unstable auto-versioning from scratch | Phase 1, step 2 |
| First stable release via `release` bump | Phase 1, step 4 |
| Tags are correct and point to right commits | Phase 1, step 4 |
| `go install @latest` works | Phase 1, step 5 |
| Build counter per calendar day | Phase 2, steps 2–3 |
| Patch bump increments correctly | Phase 2, steps 4–5 |
| Old stable release is immutable | Phase 2, step 5 |
| `go install @latest` resolves to newest stable | Phase 2, step 6 |
| Self-check integrates with the project | Phase 2, step 7 |
