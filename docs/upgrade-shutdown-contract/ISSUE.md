# Define exit-code and halt-intent contract for seid expected halts

## Problem

When seid hits a governance-scheduled upgrade height and lacks a registered handler for the plan name (i.e. an operator has not upgraded the binary), it calls Go `panic()`. There are three such panic paths in `sei-cosmos/x/upgrade/abci.go`:

| Line | Reason | Writes `upgrade-info.json` |
|---|---|---|
| 44 | Downgrade detected — last applied plan has no handler in this binary | no |
| 99 | Binary too new — has handler whose height has not arrived (non-minor upgrade) | no |
| 119 (`panicUpgradeNeeded`) | At/past upgrade height, no handler for plan | yes |

All three exit with Go's panic exit code (2). The panic also dumps a stack trace to stderr that obscures the relevant message, though the message is also raw-written to stderr explicitly to keep cosmovisor's regex scanner working (see `abci.go:114-116`).

To any process supervisor — Kubernetes (`lastState.terminated.exitCode`), systemd, our own sei-k8s-controller — these halts are indistinguishable from a genuine crash (nil deref, OOM, state-machine bug). The only structured signal today is a stderr regex.

## Customer & job-to-be-done

**Primary: sei-k8s-controller.** The controller is the supervisor we own end-to-end. When seid stops, the controller needs a machine-readable signal of *why* — to decide whether to restart in place, swap the container image, page on-call, or update the `SeiNode` CRD's status. Today it cannot tell "operator forgot to upgrade" from "state-machine bug" without log-scraping, which is brittle and racy.

**Secondary: validator operators on systemd or cosmovisor.** They have a cosmovisor regex workaround today; this gives them a cleaner, exit-code-based contract.

## Scope (v1)

**In scope** for this issue / PR:

- A new file in sei-config defining the contract: `ShutdownReason` enum, exit-code constants (70/71/72 with 73-79 reserved, 80-89 reserved for future non-upgrade graceful halts), `HaltIntent` struct (typed and JSON-serializable), `ParseExitCode` helper. ~30-50 lines plus a unit test.
- The companion design document (`DESIGN.md`) covering: producer-side change shape in sei-chain, opt-in stay-alive mode, the new optional `halt_intent` field on the existing `/status` response served by sei-tendermint RPC, and cross-repo coordination.

**Out of scope:**

- The producer-side change in sei-chain (separate issue/PR).
- The consumer-side wiring in sei-k8s-controller (separate issue/PR).
- A node-side sidecar with an aggregated `/status` adapter (Tier-2 future work).
- Persisted halt-intent files, termination-log JSON, halt-intent on EVM/Admin/Cosmos-gRPC surfaces.
- Replacing all three panic paths in one go — they can land sequentially.

## "Done" criteria

- New file in sei-config (e.g. `exitcodes.go`) with constants, `ShutdownReason`, `HaltIntent`, `ParseExitCode`, and a `String()` method on the enum.
- A unit test exercising every constant through `ParseExitCode` plus round-tripping `HaltIntent` through JSON.
- All exported symbols carry godoc that reads as a stable contract, not implementation notes.
- Numeric exit codes documented to be append-only — renumbering is a major-version break.
- Follow-up issue filed on sei-chain (producer) referenced in this PR.
- Follow-up issue filed on sei-k8s-controller (consumer) referenced in this PR.

## Note on placement

This admittedly stretches sei-config's "leaf config library" charter (`CLAUDE.md`: zero panics, no IO outside its own concerns, minimal deps). The proposed addition is constants + a typed wire shape + a pure parser — within charter. The real risk is shipping the contract before either the producer or any consumer is ready, which leaves dead code on `main`.

**Mitigation:** this PR does not merge until the two follow-up issues (sei-chain producer, sei-k8s-controller consumer) are filed and linked.

**Long-term:** if more cross-process protocol constants accumulate, they may want a dedicated home (e.g., a `sei-protocol/sei-runtime` package). For v1 the existing import graph wins — all three consumers (`seid`, `seictl`, `sei-k8s-controller`) already import sei-config.
