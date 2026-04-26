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
- A structured **`HaltIntent` value** populated before exit, exposed as an optional `halt_intent` field on the existing `/status` response served by the sei-tendermint RPC server (port 26657).

The HaltIntent shape is owned by sei-config. Every consumer — sidecar, systemd unit, controller, human curl — reads the same primitive. This keeps tier 1 universal and predictable.

**Tier 2 — node-side sidecar (control-plane adapter, future, not in scope).** A sidecar process running alongside seid in the same Pod (for the controller deployment topology) polls Tier-1 primitives and exposes an opinionated, aggregated `/status` to the control plane. This is where seid's surface-area volatility and tech debt are massaged into a stable control-plane API. **Not part of this PR; mentioned only so reviewers understand why Tier-1 stays minimal.**

The two-tier split is what reconciles "synergistic with the sidecar" with "doesn't require the sidecar." Tier 1 is sidecar-agnostic; the sidecar is purely a convenience layer.

---

## ⚠️ Decision: extend `/status` with a `halt_intent` field, not a dedicated route

This is the most contested call in the design. The recommendation flipped during PR review after an empirical audit invalidated the original premise. Reviewers should land here first.

### TL;DR

Add an **optional `halt_intent` field** to the existing `/status` response on the sei-tendermint RPC server. The field is omitted when the node is healthy and populated with a `HaltIntent` value when a graceful halt is in progress. **Do not** add a dedicated `/halt_intent` route.

### The case for extending `/status`

1. **Idiomatic.** Control-plane→dataplane state polling lives on `/status`. One endpoint, one response, one parse.
2. **Discoverability.** Sidecars and operators already poll `/status`; a new route is one more thing to know about and version-gate.
3. **Minimizes surface area on a doomed endpoint.** Sei intends to deprecate Tendermint RPC eventually. Every additional route there is a chip that has to be migrated. Extending an existing field set is no migration cost beyond what the route itself already costs.
4. **Additive JSON is non-breaking.** Adding an optional field to a JSON response is non-breaking unless consumers do strict-schema validation (`additionalProperties: false`), which is rare. Per the audit (below), no Sei consumer does this.
5. **Upstream-merge tax does not apply.** In a typical Cosmos fork, modifying `ResultStatus` would mean re-merging on every CometBFT minor release. Sei never merges sei-tendermint changes back upstream, so this is not a cost.

### What we initially argued and why it didn't hold

The first draft of this design recommended a dedicated `/halt_intent` route on **failure-domain isolation** grounds: `/status` is "the most heavily polled endpoint in the entire Cosmos universe — Prometheus exporters, Tenderduty, validator dashboards, block explorers, load-balancer health checks." Coupling halt-intent population to `/status` would risk a halt-intent bug taking down sync-status visibility for the entire monitoring fleet during the upgrade window.

That argument was load-bearing on a premise that does not hold for Sei's actual deployment stack.

### What the audit found

A thorough audit of `/status` consumers in the workspace (`sei-chain` + vendored `sei-cosmos` + `sei-tendermint`) returned only:

- `sei-tendermint/rpc/test/helpers.go:37` — `waitForRPC()` test bootstrap
- `sei-cosmos/contrib/localnet_liveness.sh:32` — liveness loop in a localnet script
- `sei-tendermint/networks/remote/integration.sh` — e2e setup curls
- `integration_test/autobahn/autobahn_test.go:71-73` — explicitly *avoids* `/status` (uses `/abci_info` instead)

**No production code in seid polls `/status`.** No Prometheus, Grafana, or Tenderduty configuration is wired to it anywhere in the workspace. Cosmovisor scans stderr, not RPC. The "monitoring fleet" the failure-isolation argument was protecting does not exist for Sei.

The single known unknown is `sei-k8s-controller`, which is in a separate repo not present in this workspace. The team's stated belief is that production `/status` consumption is "strictly integration tests and thin orchestration systems," which extends to the controller's usage. We accept this characterization for now; the consumer follow-up PR can verify.

### What's left of the failure-isolation concern

Only **defensive programming**: a bug in halt-intent population (lock contention, nil-deref on `PlanName`, serialization edge case) shouldn't be able to crash the `/status` handler. This is solvable cheaply — see the [concurrency contract](#concurrency-contract) below — and does not require a separate endpoint.

### Counterarguments and remaining concerns

- *"A nested field welded to `ResultStatus`'s lifetime is harder to migrate when we deprecate Tendermint RPC."* Mitigated by the additivity of JSON: a successor surface can simply re-emit the same field shape, or migrate consumers field-by-field. The shape is owned by sei-config and stable independent of the route that serves it.
- *"What if a future production consumer (Prometheus exporter, third-party explorer) starts polling `/status` and is sensitive to schema changes?"* The new field is additive and `omitempty`. A consumer that doesn't know about `halt_intent` sees the same shape it sees today. A consumer that *does* know about it gets the signal for free without a separate poll.
- *"Should we keep a fallback to a dedicated route if the defensive-programming approach proves insufficient?"* No — easier to add a route later than to retire one. Ship the simpler design first.

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

// HaltIntent is the structured signal seid populates on /status and that
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

1. Sets the HaltIntent on a process-global holder (an `RWMutex`-protected struct in a small new package, importable by both `x/upgrade` and the `/status` handler in `sei-tendermint`).
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

## `halt_intent` field on `/status` (also out of scope for this PR, sketched)

The existing `/status` handler in `sei-tendermint` is extended to populate an optional `halt_intent` field on its response, sourced from the in-process holder. The handler change is contained — a few lines reading from the holder and adding the field to the existing `ResultStatus`-shaped response.

### Response shape

- **Healthy node:** `halt_intent` is omitted (or null, depending on JSON-encoding choice — `omitempty` recommended). Existing `/status` consumers see no change.
- **Halted node:** `halt_intent` is populated with the full `HaltIntent` value defined in sei-config.

This means a `/status` consumer that has never heard of `halt_intent` continues to work; one that knows the field gets the halt signal for free without polling a second endpoint.

### Concurrency contract

This is the only piece of the original design that survives unchanged — it's the cheap defensive programming that absorbs what's left of the failure-isolation concern.

- HaltIntent state is held under an `RWMutex` in a small holder package importable by both `x/upgrade` (writer) and `sei-tendermint` (reader).
- The producer (BeginBlocker → `gracefulHalt`) writes once under `Lock()` and never mutates the value after. It is safe to publish a pointer once and never replace it.
- The reader (the `/status` handler's halt-intent lookup) takes `RLock()`, copies a snapshot out of the lock, serializes outside.
- **The halt-intent population path inside the `/status` handler must `defer recover()`** so that a bug in halt-intent code (nil-deref, serialization panic, lock misuse) cannot 500 the broader `/status` response. On recover, log and emit `halt_intent: null`. `/status` continues to serve sync state.

### Discoverability

`/status` is already the canonical place to ask seid "what's your state?". No new advertisement, no `node_info.other` entry, no version-gating dance. Consumers that know the field use it; consumers that don't keep working as before.

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
- **Follow-up sei-chain issue/PR:** implements the producer (graceful-halt helper, stay-alive mode flag, `halt_intent` field on the `/status` response in sei-tendermint).
- **Follow-up sei-k8s-controller issue/PR:** implements the consumer (poll `/status`, branch on the `halt_intent` field's `reason`, drive image swap or page).

The sei-config PR must not merge until both follow-up issues are filed and linked. The producer must not merge until consumers can pin the matching sei-config version. The `/status` field extension can ship in sei-tendermint independently of the panic-replacement, but they should ship together to avoid releasing a field that is always absent because the producer never populates it.
