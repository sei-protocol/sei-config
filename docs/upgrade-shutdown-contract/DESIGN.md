# Upgrade shutdown contract — design

> Companion to [ISSUE.md](./ISSUE.md). This doc covers the design intent for a graceful, distinguishable shutdown signal when seid detects it is running an incompatible binary for an on-chain upgrade. The deliverable in *this* PR is only the sei-config-side contract; the producer (sei-chain) and consumer (sei-k8s-controller) changes are described here so reviewers can sanity-check the shape, but they ship as separate PRs.

## Background

When the x/upgrade module's `BeginBlocker` decides the running binary cannot proceed, it calls `panic()`. Three panic sites in `sei-cosmos/x/upgrade/abci.go` (lines 44, 99, 119). All exit with Go's panic code (2) and dump a stack trace; the relevant message is also raw-written to stderr for cosmovisor regex compat.

For comparison, **the existing `--halt-height` operator path is already graceful.** `BaseApp.halt()` at `sei-cosmos/baseapp/abci.go:288-321` does *not* panic — it sends `SIGINT` to itself, which the existing shutdown defer in `server/start.go:459-484` catches and tears everything down cleanly. The upgrade halt is the outlier; this design proposes bringing it in line.

## Goals

- A stable, machine-readable signal that distinguishes "operator action required" halts from genuine crashes.
- The signal is observable both **post-mortem** (process exit code) and, optionally, **live** (status endpoint, while the process is still running) so a control plane can fetch *why* the node is no longer producing blocks.
- Sidecar-optional. External operators (validators on systemd, humans with curl) get the same primitive.
- Backwards-compatible default. Today's "the process exits when it can't proceed" behavior is preserved; the live-status mode is opt-in via a flag.

## Non-goals

- Replace cosmovisor or systemd integration paths.
- Define a unified node-status aggregator API. That's an explicit *Tier-2 future*; see [Architecture](#architecture-two-tier).
- Solve binary-image lookup, restart policy, image swap, or CRD modeling — those belong to the consumer (sei-k8s-controller).
- Modify the existing `upgrade-info.json` artifact (cosmovisor's contract). It continues to be written exactly as today; we do not extend its schema.

## Architecture (two tier)

**Tier 1 — seid (dataplane primitive).** Each halt cause has:
- A distinct **exit code** in the range 70–79.
- A structured **`HaltIntent` value** populated before exit, exposed via a discrete `/halt_intent` route on the existing sei-tendermint RPC server (port 26657).

The HaltIntent shape is owned by sei-config. Every consumer — sidecar, systemd unit, controller, human curl — reads the same primitive. This keeps tier 1 universal and predictable.

**Tier 2 — node-side sidecar (control-plane adapter, future, not in scope).** A sidecar process running alongside seid in the same Pod (for the controller deployment topology) polls Tier-1 primitives and exposes an opinionated, aggregated `/status` to the control plane. This is where seid's surface-area volatility and tech debt are massaged into a stable control-plane API. **Not part of this PR; mentioned only so reviewers understand why Tier-1 stays minimal.**

The two-tier split is what reconciles "synergistic with the sidecar" with "doesn't require the sidecar." Tier 1 is sidecar-agnostic; the sidecar is purely a convenience layer.

---

## ⚠️ Decision: dedicated `/halt_intent` route, NOT extending `/status`

This is the most contested call in the design. Reviewers will land here first — calling it out explicitly with the case for and against. Push back hard on this section if you disagree.

### TL;DR

Add a **new dedicated route `/halt_intent`** on the sei-tendermint RPC server. Do **not** extend the existing `/status` response to include a `halt_intent` field.

### Why this is non-obvious

`/status` is the idiomatic place a control-plane fetches a dataplane process's state. Most reviewers' first instinct — and the original direction of this design — was to extend `/status` with an optional `halt_intent` field that's null when healthy. The "new endpoint" feels like a smell, especially since both routes live on a surface (Tendermint RPC) that we want to deprecate eventually.

### The case for extending `/status` (rejected)

1. **Idiomatic.** Control-plane→dataplane state polling lives on `/status`. One endpoint, one response, one parse.
2. **Discoverability.** Sidecars and operators already poll `/status`; a new route is one more thing to know.
3. **No new surface to deprecate later.** If we eventually retire Tendermint RPC entirely, every additional route there is a chip we have to migrate.
4. **Upstream-merge tax** *(does not apply for Sei — see below)*. In a typical Cosmos fork, modifying upstream-defined types like `ResultStatus` means re-merging on every CometBFT minor release. **For Sei this argument is moot:** Sei never merges sei-tendermint changes back to upstream; the fork is fully owned. We can modify `ResultStatus` freely.

### Why we still recommend a dedicated route (failure-isolation)

Of the arguments above, only #4 was decisive in the original analysis, and Sei's never-merge-upstream posture neutralizes it. So why still go with a dedicated route?

**Failure-domain isolation.** This is the strongest single argument and the one most reviewers don't anticipate.

`/status` is the most heavily polled endpoint in the entire Cosmos universe — Prometheus exporters, Tenderduty, validator dashboards, block explorers, load-balancer health checks, the sei-k8s-controller's existing health probes. Coupling halt-intent population to `/status`'s critical path means any bug in the halt-intent reader — lock contention with the consensus shutdown path, a nil-deref on `PlanName`, a serialization edge case — takes down sync-status visibility for the entire monitoring fleet.

Crucially, that breakage would happen **exactly during the upgrade window** when operators most need `/status` to be boring and reliable. We would be introducing a new failure mode at the worst possible time.

A dedicated `/halt_intent` handler with its own mutex and its own `defer recover()` keeps the blast radius scoped: a bug in halt-intent code can break `/halt_intent` only, and `/status` keeps reporting sync state to everyone polling it.

### Secondary argument (deprecation)

The "we want to deprecate Tendermint RPC eventually" point cuts the *opposite* way from initial intuition. A discrete route is trivially shimmed, proxied, or replaced by whatever succeeds Tendermint RPC. A nested field inside `ResultStatus` is welded to the struct's lifetime — a future replacement has to keep emitting the same field at the same nesting depth, or migrate every consumer in lockstep. The dedicated route gives us more freedom, not less.

### Counterarguments and how we address them

- *"A new endpoint is a smell."* Real, but the smell is preferable to the failure-isolation cost. The endpoint is small (one method, one response shape, one mutex) and serves a single purpose. It is not a kitchen-sink endpoint waiting to grow.
- *"Discoverability."* The Tier-2 sidecar (when it ships) knows about `/halt_intent` natively and is the controller's primary path. For systemd/curl operators, discoverability is solved by docs. We deliberately do **not** advertise `/halt_intent` via `/status`'s `node_info.other` map — that re-couples the two routes' release lifecycles, which the failure-isolation argument is trying to decouple.
- *"What about the deprecation."* See above. A dedicated route is *easier* to migrate, not harder.

### Open question (defer)

If a future contributor argues that the failure-isolation risk is overstated — i.e. demonstrates that the halt-intent reader can be made unconditionally safe with no shared lock with consensus — the right move is *still* a dedicated route on first ship, and revisit consolidation later. Easier to merge two routes than split one.

---

## sei-config contract (v1 deliverable in this PR)

A new file (proposed name: `exitcodes.go`):

```go
package seiconfig

import "time"

// ShutdownReason identifies why seid halted. The numeric values are stable
// process exit codes; see the ExitCode* constants below. Values are append-only —
// renumbering any constant is a backwards-incompatible change.
type ShutdownReason int

const (
    ShutdownReasonUnknown            ShutdownReason = 0
    ShutdownReasonUpgradeRequired    ShutdownReason = 70
    ShutdownReasonBinaryTooNew       ShutdownReason = 71
    ShutdownReasonDowngradeDetected  ShutdownReason = 72
    // 73-79 reserved for future upgrade-related graceful halts.
    // 80-89 reserved for non-upgrade operator-action halts (e.g. halt-height).
)

// Exit codes used by seid to signal graceful halts to process supervisors.
// Identical numeric values to the corresponding ShutdownReason; duplicated as
// distinct constants because process supervisors see exit codes while in-process
// code uses the typed enum.
const (
    ExitCodeUpgradeRequired   = 70
    ExitCodeBinaryTooNew      = 71
    ExitCodeDowngradeDetected = 72
)

// HaltIntent is the structured signal seid serves on /halt_intent and that
// describes the reason for an in-progress or pending graceful halt.
//
// JSON tags are part of the wire contract and must not be renamed without
// coordinated consumer updates.
type HaltIntent struct {
    Reason       ShutdownReason `json:"reason"`
    ReasonString string         `json:"reason_string"`        // human-readable, fixed phrasing per Reason
    PlanName     string         `json:"plan_name,omitempty"`
    Height       int64          `json:"height,omitempty"`
    Info         string         `json:"info,omitempty"`
    AnnouncedAt  time.Time      `json:"announced_at"`
}

// ParseExitCode maps a process exit code to a ShutdownReason. Returns
// (ShutdownReasonUnknown, false) for codes outside the graceful range.
func ParseExitCode(code int) (ShutdownReason, bool) { ... }

// String returns a stable lowercase token suitable for logs and the
// HaltIntent.ReasonString field.
func (r ShutdownReason) String() string { ... }
```

That's the entire library surface. Plus a unit test exercising every constant through `ParseExitCode` and round-tripping `HaltIntent` through `encoding/json`.

Notable omissions, all deliberate:
- **No `UpgradeInfo` reader.** The on-disk `upgrade-info.json` shape is owned by `x/upgrade/keeper` in sei-cosmos. Duplicating it here would create a drift hazard with no payoff — sei-config doesn't need it; the controller can vendor a 30-line struct itself if it ever does.
- **No file IO.** sei-config does not write or read `halt-intent.json`. The single source of truth is the live endpoint.
- **No constructor for the SIGINT/exit helper.** The producer-side helper (`gracefulHalt`) lives next to its callsites in sei-cosmos, since it needs the upgrade keeper for `DumpUpgradeInfoWithInfoToDisk`. It references `seiconfig.ExitCodeUpgradeRequired` etc. The library does not export a `GracefulHalt(...)` function.

## Producer side (sei-chain — out of scope for this PR, sketched)

In `sei-cosmos/x/upgrade/abci.go`, replace the three `panic()` calls with calls to a new helper `gracefulHalt(reason, intent)` that:

1. Sets the HaltIntent on a process-global holder (an `RWMutex`-protected struct in a small new package, importable by both `x/upgrade` and the `/halt_intent` HTTP handler in `sei-tendermint`).
2. Performs the existing logging + raw stderr write + (for the line-119 path) `upgrade-info.json` write. **No change to those existing artifacts.**
3. Sends `SIGINT` to self, mirroring the existing `BaseApp.halt()` pattern.
4. The shutdown defer in `sei-cosmos/server/start.go:459-484` is extended to read the holder, look up the matching exit code via `seiconfig.ParseExitCode`, and call `os.Exit(N)` with the distinct code instead of returning normally.

This reuses the existing graceful-shutdown plumbing rather than inventing a parallel one. It is a much smaller change than it would first appear.

### Default vs stay-alive mode

A new boolean flag, default `false`, gates the live-endpoint behavior:

- **Off (default — equivalent to today):** `gracefulHalt` triggers the SIGINT shutdown path immediately. Process exits with the distinct exit code. Status servers go down with consensus. No surprise for current operators; the only observable change vs. today is a clean exit code (70/71/72) instead of panic exit code 2 with a stack trace.
- **On (stay-alive):** `gracefulHalt` populates the holder, stops consensus, and **leaves the four servers running indefinitely**. Process exits only on external `SIGTERM`/`SIGINT`. The supervisor (sidecar, controller, human) drives termination after observing the halt intent and taking action.

Stay-alive mode requires splitting the shared `goCtx` in `sei-cosmos/server/start.go` so consensus and the servers have separate cancellation. This is real surgery but contained to that file.

A flag name is deliberately not specified in this design — the producer PR can pick something like `--halt-stay-alive` or `--halt-stays-running`. Naming bikeshed deferred.

## /halt_intent route (also out of scope for this PR, sketched)

A new Tendermint RPC route on `sei-tendermint`, registered alongside the existing Sei-only `/lag_status`. Path: `/halt_intent`. Always reachable while the RPC server is up.

### Response

- Always **`200 OK`** with the HaltIntent JSON.
- When healthy: `{"reason": 0, "reason_string": "healthy", "announced_at": "..."}`.
- When halted: the populated HaltIntent.

**Not 204. Not 404.** A 404 is indistinguishable from "older seid doesn't have this route" and breaks consumer version-probing. We always answer 200 with a populated `reason` field.

### Concurrency contract

- HaltIntent state is held under an `RWMutex` in the holder package.
- The producer (BeginBlocker → gracefulHalt) writes once under `Lock()` and never mutates the value after. It is safe to publish a pointer once and never replace it.
- The handler reads under `RLock()`. It copies a snapshot out of the lock and serializes outside.
- The handler has a `defer recover()` so a bug in halt-intent serialization cannot bring down the broader Tendermint RPC server (failure-isolation principle from the [decision section](#-decision-dedicated-halt_intent-route-not-extending-status)).

### Discoverability

The Tier-2 sidecar knows about `/halt_intent` natively. For non-sidecar consumers (systemd, curl), the route is documented in seid's reference docs. We do **not** advertise it via `/status`'s `node_info.other` field — that would re-couple the two routes' lifecycles, which contradicts the failure-isolation rationale for splitting them.

## What's NOT in this PR (or this design's scope)

- The producer-side sei-chain change. Tracked as a follow-up issue (must be filed before this PR merges).
- The consumer-side sei-k8s-controller wiring. Tracked as a follow-up issue (same).
- The Tier-2 sidecar `/status` aggregator. Future work.
- Halt-intent on EVM RPC, Admin gRPC, or Cosmos gRPC surfaces. Tier 1 has one canonical surface (Tendermint RPC). Multiple surfaces re-introduce the version-coordination problem the dedicated route is designed to avoid.
- A persisted `halt-intent.json` file. The live endpoint is the single source of truth; on-disk state can desync.
- A termination-log JSON write to `/dev/termination-log`. Was considered (kubelet surfaces it via `lastState.terminated.message`); deferred — adds another contract surface for the controller follow-up to absorb without solving anything that the live endpoint doesn't.
- Replacing all three panic paths atomically. They can land sequentially in the producer PR. The line-119 path (most common case — operator forgot to upgrade) is the natural first target.

## Open questions

1. **Cosmovisor coexistence.** If validator pods run cosmovisor inside the container and we ship stay-alive mode, does cosmovisor swallow our exit code via its own restart loop? For controller-managed pods the controller likely bypasses cosmovisor entirely; needs confirming when the producer PR lands.
2. **Holder location.** The HaltIntent holder needs to be readable by the Tendermint RPC handler (in `sei-tendermint`) and writable by the upgrade keeper (in `sei-cosmos`). Concretely: a small new package both import? Or hang it off an existing shared type? Producer-side detail; defer to that PR.
3. **Image-mapping source for sei-k8s-controller.** When the controller observes `Reason=70 PlanName=v6`, where does it look up "v6 → ghcr.io/sei/seid:v6.0.1"? CRD field, ConfigMap, annotation? Out of scope here; flagged for the consumer follow-up.

## Cross-repo coordination

- **This PR (sei-config):** ships the contract.
- **Follow-up sei-chain issue/PR:** implements the producer (graceful-halt helper, stay-alive mode flag, `/halt_intent` route on sei-tendermint).
- **Follow-up sei-k8s-controller issue/PR:** implements the consumer (poll `/halt_intent`, branch on `ShutdownReason`, drive image swap or page).

The sei-config PR must not merge until both follow-up issues are filed and linked. The producer must not merge until consumers can pin the matching sei-config version. The `/halt_intent` route can ship in sei-tendermint independently of the panic-replacement, but they should ship together to avoid releasing a route that always reports "healthy" for halt conditions a binary cannot otherwise survive.
