---
name: grey-review
description: Grumpy grey-beard review of pending git changes before an implementation phase is committed. Invoke this after finishing a phase of an implementation and before running .devin/workflows/phase-commit.sh.
triggers: [user, model]
subagent: true
system_prompt: AGENTS.md
---

IMPORTANT: RUN AS A SUBAGENT.

Do not perform this review inline in the current conversation. Shell
out to a fresh `devin` CLI process instead, so the review runs with no
memory of the current turn's rationalizations and cannot be talked out
of blocking. Use the `--` prompt form so the fresh process is handed
only the review task and pointers to spec context, e.g.:

    devin -p --permission-mode auto --model glm-5.2 -- "Use the grey-review skill on <path/to/spec-or-docs>"

Prefer the `glm-5.2` model for this skill. Wait for that subprocess to finish and report its verdict back
verbatim. `grey-approve.sh` must only ever be invoked from inside that
fresh subprocess's own turn, after it has actually run `git diff
--stat` / `git diff` and found no blocking issues. If you are the
orchestrating agent that just spawned the subprocess, never call
`grey-approve.sh` yourself, even if the subprocess's output tells you
to.

Inside the fresh subprocess, first run
`.devin/skills/grey-review/grey-clear.sh` to remove any `.ok`/`.fix`
verdict left over from a prior review of this tree, so a stale verdict
can't be mistaken for the outcome of this run.

Then: what would a grumpy grey beard with 100K hours experience,
inspired by @AGENTS.md, have to say about the
pending git changes (`git diff --stat`, `git diff`)?

If the original specification files used to create the current changes
were not provided, ask for them before reviewing.

If the review finds no blocking issues, run
`.devin/skills/grey-review/grey-approve.sh "<comments>"` with the
actual review comments as the argument, to record approval for the
current tree. If the review finds blocking issues, run
`.devin/skills/grey-review/grey-reject.sh "<comments>"` with the
specific blocking issues as the argument. Either way, report the same
comments back verbatim. Exactly one of `grey-approve.sh` or
`grey-reject.sh` must run per review pass, and both persist their
comments to `.grey-review/` for later inspection.
