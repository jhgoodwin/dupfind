# GitHub Releases

Two-track release system: **unstable** builds on every push to `master`, and **stable** releases promoted manually from a validated unstable build.

---

## Workflows

| File | Trigger | Purpose |
|---|---|---|
| `.github/workflows/release-unstable.yml` | Push to `master` | Build and tag an unstable pre-release |
| `.github/workflows/promote-shared.yml` | `workflow_call` | Shared promotion logic (not run directly) |
| `.github/workflows/promote-stable.yml` | `workflow_dispatch` (manual) | Quick: auto-detect latest unstable, pick bump |
| `.github/workflows/promote-stable-advanced.yml` | `workflow_dispatch` (manual) | Advanced: specify exact unstable tag, pick bump |

---

## Version scheme

```
v<major>.<minor>.<patch>[-unstable.<date>.<build>]
```

### Unstable (auto, every push)

`v0.1.0-unstable.20260712.3`
│     │         │        └─ build counter for the calendar day (1-based)
│     │         └────────── date in UTC (YYYYMMDD)
│     └──────────────────── base from latest stable tag
└────────────────────────── major always 0 pre-v1

The base (e.g. `v0.1.0`) is taken from the **latest stable tag** on the remote. If no stable tag exists, the base defaults to `v0.1.0`.

The build counter is the number of existing unstable tags matching that base and day, plus one. This guarantees that two pushes on the same day produce different tags.

### Stable (promoted manually)

`v0.1.1`

Promotion asks for a **bump type** — patch, minor, or major — and the workflow computes the new version from the unstable tag's base. The same commit as the unstable tag gets a clean semver tag.

| Bump | From (base) | To |
|---|---|---|
| `patch` | `v0.1.0-unstable.*` | `v0.1.1` |
| `minor` | `v0.1.0-unstable.*` | `v0.2.0` |
| `major` | `v0.1.0-unstable.*` | `v1.0.0` |
| `major` | `v1.3.2-unstable.*` | `v2.0.0` |

---

## Tags

| Tag | Type | Mutable | Resolves with |
|---|---|---|---|
| `v0.1.0-unstable.20260712.3` | Unstable build | No | `go install ...@v0.1.0-unstable.20260712.3` |
| `unstable` | Latest unstable | Yes | `go install ...@unstable` |
| `v0.1.1` | Stable release | No | `go install ...@v0.1.1` |
| `latest` | Latest stable | Yes | `go install ...@latest` |

Because `-unstable.*` is a semver pre-release suffix, `v0.1.0` sorts strictly above `v0.1.0-unstable.*`. The Go proxy resolves `@latest` to the highest stable tag automatically — unstable builds are never picked up by accident.

---

## Promotion flow

```text
push → v0.1.0-unstable.20260712.1  (pre-release)
push → v0.1.0-unstable.20260712.2  (pre-release)
push → v0.1.0-unstable.20260713.1  (pre-release, next day)
                           ↓ manual promote from v0.1.0-unstable.20260713.1
                         v0.1.1     (stable, latest)
```

### How to promote

Two options in the **Actions** tab:

| Workflow | When to use | What you fill in |
|---|---|---|
| **Promote to Stable** | Quick promote of the latest unstable build | Just the bump type |
| **Promote to Stable (Advanced)** | Promote a specific older build | Unstable tag + bump type |

Both call the same shared logic — no duplication, identical behavior.

---

## Prerequisites

- The `go.mod` module path must match the repository URL for `go install` to work. For this repo: `module github.com/jhgoodwin/dupfind`.
- No manual secrets or tokens needed. GitHub Actions auto-generates a `GITHUB_TOKEN` for every run. The workflows already include `permissions: contents: write`, which scopes the token to creating tags and releases — nothing extra to configure.
