# AGENT.md

Goal: build Cheatr MVP from `PLAN.md`.

## Ground Rules

### DO
- Use `PLAN.md` as source of truth.
- Execute `## Implementation Plan (Todo) -> MVP (v1)` in exact numeric order.
- Spawn one subagent per MVP item.
- Keep subagent scope to that one item only.
- After item done: validate, commit, then move to next.
- Keep code simple/readable. Small funcs. clear names. early returns.
- If logic is hard/non-obvious, add short comments (why/invariant).

### DO NOT
- Do not skip/reorder MVP items.
- Do not work on `Future` items unless explicitly asked.
- Do not mix multiple MVP items in one commit.
- Do not add speculative abstractions for "later".
- Do not use fuzzy behavior when `PLAN.md` says strict.
- Do not keep more than one task "in progress".

## Subagent Concurrency (HARD)

- Max 1 active subagent at any time.
- Never spawn a second subagent while one is active.
- No parallel subagent runs. ever.
- If blocked: stop, report blocker, do not spawn another subagent.

Enforcement:
1. Track one `active_subagent_task_id`.
2. Before spawn: if set, do not spawn.
3. On completion: validate -> commit -> clear id.
4. Then start next item.

## Per-Item Workflow

For item `N` in MVP list:
1. Read item `N` in `PLAN.md`.
2. Spawn exactly one subagent for item `N`.
3. Implement only item `N`.
4. Run checks relevant to item (`go test ./...` when available).
5. Commit immediately.
6. Log: `ITEM N DONE`, commit hash, brief notes, next item.

If incomplete/blocker:
- Log `ITEM N BLOCKED` with reason + required input.
- Do not continue to `N+1`.

## Commit Policy

### DO
- One commit per MVP item.
- Keep commit scoped and atomic.
- Suggested message: `feat(step-N): <short outcome>`.

### DO NOT
- No amend unless explicitly asked.
- No force push.
- No committing secrets.

## Quality Bar

### DO
- Prefer straightforward over clever.
- Keep side effects explicit (network/cache/fs).
- Keep routing + keymap deterministic.
- Add tests for non-trivial resolver/search behavior.

### DO NOT
- No giant functions.
- No hidden global mutable state.
- No dead code/TODO without context.
