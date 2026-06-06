# Configuration manager — design

> Refined after cross-repo validation against the live `sei-chain`, `sei-k8s-controller`, and the `seictl` sidecar. Phase vocabulary from [CLAUDE.md](../../CLAUDE.md): Phase 2 = today's two-file layout; Phase 3 = unified `sei.toml`. Implementation lands as separate PRs (sei-config + sei-chain + seictl).

## Background

A `seid` node's config is spread across `config.toml` (Tendermint), `app.toml` (Cosmos + Sei sections: `evm`, `state-store`, `giga_executor`, …), `client.toml`, cobra flags, and `SEID_*`/`SEI_*` env vars resolved by Viper — loaded in `PersistentPreRunE` (`root.go:79-104` → `InterceptConfigsPreRunHandler` → `interceptConfigs`, in the vendored `sei-cosmos` fork). The **sei-config library already exists** (unified `SeiConfig`, `DefaultForMode()`, `Validate()`, a key→env→file registry, `SEI_*`/`SEID_*` resolution, a `MigrationRegistry` at `CurrentVersion=2` with one shipped v1→v2 migration, atomic two-file IO). This project makes `seid` a **consumer**: a second, *experimental* configuration manager that `seid` selects over the legacy loader when an experimental env var says so.

**Where the risk actually lives.** It is at the selection seam **and in the library's legacy round-trip fidelity** — the latter is not free. The clean mental model: **the registry/unified key space is a strict superset of the keys the legacy two-file IO round-trips, and that difference is a silent-drop surface.** `toml.DecodeFile` ignores unknown keys and the encoder writes only modeled fields, so any real `seid` key the legacy mapper doesn't model is dropped — or, on a tag mismatch, corrupted — on a read→write, silently. That set is non-empty today (see MVP). "The library is the asset" holds, with this asterisk.

## Goals

- `seid` **selects** its configuration manager at startup via an experimental env var — legacy loader (default) vs the sei-config-backed manager; legacy path byte-for-byte unchanged.
- The manager exposes the library in-binary (`doctor` / `generate --mode` / `migrate`).
- A versioning contract **keyed on seid release** that migrates config safely across releases.
- All `seid`-side changes **inside the sei-chain repo** — zero lines of the `sei-cosmos` fork.

## Non-goals

Phase 3 (`sei.toml`); owning `seid init`, `client.toml`, secrets, or hot-reload; **day-2 config mutation** — running-node changes go through the controller's `ConfigPatchTask` raw-TOML merge, which bypasses the library entirely; bringing that under sei-config is a *named follow-up*, not this scope; folding sei-config into sei-chain *now*; mode extensions (dedicated `replayer`, `seed`/CRD reconciliation — [Appendix A](#appendix-a--modes-beyond-the-core)).

## ⚠️ Decision: the experimental manager + user CLI live in `seid`

The sei-config-backed manager — including the `config …` surface (`doctor`/`generate`/`migrate`/`show`) — runs **inside the `seid` binary** (urfave/cli v3 under a cobra delegate). Putting the *user-facing CLI* in the node binary means the CLI's view of config cannot drift from the node's. Seam constraint: **`config` must skip `PersistentPreRunE`** — that hook runs for every subcommand, so without a guard `seid config …` would trigger the legacy interception (and under `SEI_CONFIG_MANAGER=v2`, the gated seam *recursively*), mutating files just to inspect them. Extend the `init` skip at `root.go:97` to also short-circuit `config`. Delegation + costs: [Appendix B](#appendix-b--seid-config-cobraurfave-integration).

> **⚠️ This does *not* eliminate library-version skew — a `seictl` sidecar is already the production config writer.** The controller does **not** call sei-config directly: it builds a `ConfigIntent` and ships it over HTTP to a **`seictl` sidecar** (`seictl@v0.0.55`) in the node pod, which runs `ResolveIntent`/`WriteConfigToDir` onto the shared PVC `seid` reads. So config is written by the *sidecar's* pinned sei-config and read by the *seid binary's* pinned sei-config — two independently-versioned copies touch the same files. The in-binary user CLI removes *CLI*↔node skew; it does **not** remove *sidecar*↔*seid* library skew. See Architecture and Versioning.

## Architecture — four components, three sei-config pins

| Piece | Repo / image | Role | sei-config pin |
|---|---|---|---|
| `seiconfig` library | sei-config | Resolution, modes, validate, migrate, legacy IO | — (it *is* the lib) |
| Gated manager + reader + CLI | sei-chain (`seid`) | Selects legacy vs sei-config; reads the two files at boot; hosts the user CLI | seid go.mod |
| Intent producer | sei-k8s-controller | Builds `ConfigIntent` from the CRD; **does not write config itself** | controller go.mod |
| Intent resolver / writer | `seictl` sidecar | Receives intent over HTTP; `ResolveIntent`→`WriteConfigToDir` onto the shared PVC | **sidecar image** |

**On-disk fidelity is determined by the sidecar's pin, not the controller's** — bumping sei-config in the controller's go.mod does nothing to what's written. The sidecar image drifts independently from the seid image today (`sidecarImageDrifted` is tracked separately). **Must-do:** pin the sidecar image to the seid image (or have `seid` validate what the sidecar wrote — see Versioning), else two differently-pinned libraries write/read the same files and fidelity is undefined.

**Future (deferred — but now load-bearing for *correctness*, not just cleanup).** (1) **fold-in** — sei-config collapses into the sei-chain tree (`git mv` + import change; one-way arrow sei-chain → sei-config; seam unchanged). (2) **consolidation** — the controller drives config *through `seid`* so the resolver and the node share one binary/lib, **collapsing the three pins to one**. The latter (consolidation) is the real fix for write-path skew.

**The seam contract (load-bearing).** Inject at `root.go:101-103`, after the `init` skip. `start.go`/`newApp` read config through **two** channels: `serverCtx.Config` (a `SetRoot`/`ValidateBasic`-passing `*tmcfg.Config`) and `serverCtx.Viper` (`AppOptions` — every Sei section via `appOpts.Get("evm.http_port")`). The gated path **materializes the two legacy files (to an ephemeral/derived target — see below), then re-enters the same Viper read+merge tail** (`util.go:162-219, 317-323`) + `bindFlags` for flag>env>file precedence — it must **not** feed `app.New` from the in-memory struct.

> **⚠️ Fidelity is bounded by `legacy.go` coverage, not by the read channel.** Materialization runs through `toLegacyApp()`, which encodes only modeled fields — so an unmodeled key is dropped *at materialization*, and re-entering Viper then reads a file already missing it. Re-entering Viper does not by itself deliver the safety property. The seam is sound only once the legacy mapper covers every real key (MVP fidelity test) **or** via passthrough overlay: read the operator's *existing* files into Viper first, overlay only library-owned keys, never marshal a fresh file from the struct.
>
> **⚠️ Hand-edit safety (invariant).** The materialize round-trip must be **byte-stable for keys the library doesn't model and preserve comments** — an unchanged input yields an unchanged file. If that can't be guaranteed, materialize to a derived path (`${home}/config/.managed/`) and feed Viper from there, leaving the operator's authored files as the untouched source of truth. "Hand-edits win" until legacy removal.
>
> **⚠️ Materialization is ephemeral — the boot path never persists (reconciles the seam with "no auto-*rewrite*").** The gated boot path materializes to an ephemeral/derived target (e.g. `${home}/config/.managed/`, or a temp dir Viper is pointed at), **never** the operator's authored `config.toml`/`app.toml`, which stay the read-source-of-truth. In particular, when an older on-disk `schema_version` is migrated in memory (versioning slice), the migrated result is materialized **for Viper only and never written back** — only `seid config migrate --write` persists a migration. This is what makes "materialize the files" and "no boot-time disk rewrite / downgrade-safe" non-contradictory; a build that writes migrated config to the authored files on boot is a bug.

## Env-var gate contract

`SEI_CONFIG_MANAGER` (experimental, opt-in), **value-based**: `v2` → the sei-config manager; unset/`legacy` → legacy (default); anything else → hard startup error (never silent fallback). Read via raw `os.Getenv` atop `PersistentPreRunE`; clean two-way door. Precedence (low→high): mode default < file < `SEI_*` env < flag; `SEI_*` beats deprecated `SEID_*` (stderr warning).

**`SEI_` collision audit (must-fix), refined.** `seid` claims `SEI_` via `WithViper("SEI")` (`root.go:74`) and the library uses `SEI_` too. *Under the seam, env is resolved by Viper's `AutomaticEnv`, not the library's `ResolveEnv`* — so at runtime the library's env scheme is moot; but `ResolveEnv` **is** live in the CLI/controller paths. Viper's name scheme (`SEID_MINIMUM_GAS_PRICES`, derived from flags/dotted keys) and the library's (`SEI_CHAIN_MIN_GAS_PRICES`, derived from unified struct tags) **build names by different rules and barely overlap lexically**. So the audit must assert **per-field destination-equivalence** and flag **dual-reachable** settings (one setting reachable under two env names, possibly with divergent precedence) — *not* merely diff matching `SEI_*` strings, which goes green while the real hazard hides.

## Versioning & migration — *proposed scheme, a SEPARATE slice from the seam MVP*

> **Status: to build, not shipped.** Today the library is **integer-keyed**: `CurrentVersion int = 2` (`config.go:11`), `SeiConfig.Version int`, `Migration.From/ToVersion int` with one shipped `v1→v2` WriteMode-rename entry (`migrate.go`). The controller does **not** set `ConfigIntent.TargetVersion` (it defaults to `CurrentVersion`). The scheme below replaces that. It is **larger than the seam and not on the seam MVP's critical path** — and because no live config carries the integer version (the manager hasn't rolled out), it is a **clean slate with no on-disk shim**. Sequence it as its own slice; trigger: the first config-shape change that requires a new migration. The MVP needs only refuse-on-*unmodeled-key* (no version stamp); refuse-on-*newer-version* belongs to this slice and depends on OQ2.

**Proposed.** `schema_version` becomes a **semver string tracking the seid release in which config shape last changed** (e.g. `v6.5.0`), advancing **only on a shape change** (so `6.4.1` and `6.4.2` share a version — reconciling "track seid release" with "bump only on shape change"). Comparison uses **`golang.org/x/mod/semver`** — the convention sei-chain adopted in [PR #3153](https://github.com/sei-protocol/sei-chain/pull/3153) to fix the lexical-sort bug ([cosmos-sdk#11707](https://github.com/cosmos/cosmos-sdk/issues/11707): natural string order puts `v0.9` after `v0.10` → wrong upgrade resolved → consensus panic). `x/mod` is a *direct* dep of sei-chain but *indirect* in sei-k8s-controller — the controller must promote it to a direct require if it is to compare versions.

Because **we author schema versions**, gate every value with `semver.IsValid()` and **fail loud** — a malformed `schema_version` is a `SeverityError`; never let `Compare` silently coerce an invalid into "oldest" and fire a wrong migration. Strip prerelease (`-nightly`/git-describe) before shape comparison. The change: `CurrentVersion int = 2` → `CurrentSchemaVersion = "v6.5.0"`; `SeiConfig.Version int` → `string`; the registry re-keys on `IntroducedIn`/`PreviousVersion` and applies every migration whose `IntroducedIn ∈ (on-disk, CurrentSchemaVersion]` (the shipped `v1→v2` rename becomes `SchemaBaseline → v6.5.0`).

Boot rule (direction-keyed): **older** → start, migrate **in-memory only**, no disk rewrite, WARN + metric; **equal** → start; **newer** → refuse to start. The principle is **"no auto-*rewrite*"** — the hazard is the disk write, not the transform; `seid config migrate --write` (dry-run default, `.bak` first) is the only path that *persists* an upgrade.

- **Stable-vs-nightly (seid v6.5.1, sei-chain PR #27) is a *defaults* difference, not a shape difference** — `cosmos_only` vs `memiavl_only` are both `WriteMode` enum members; it lives in `DefaultForMode`, never a separate schema version. (Note: the current default `cosmos_only` is itself a *v1* WriteMode value required by v6.5.1 stable even though `CurrentSchemaVersion` would be `v6.5.0`; the rename migration only fires on a stamped on-disk v1.) A guard test forbids two channels needing structurally different field sets under one version.
- **Controller coupling (to build):** the controller *should* derive `ConfigIntent.TargetVersion` from the pod's seid **image tag** so skew detection is accurate against the namespace the node boots under. It does **not** today. Unparseable tags (`latest`, digest pin) omit it and fall back.
- **Wire-format coordination:** changing `ConfigIntent.TargetVersion` from `int` to a semver `string` alters the controller↔sidecar `ConfigApplyTask` JSON. Land it behind the same coordinated pin-move (controller + sidecar together), **or** keep `TargetVersion` an `int` and carry semver only at the seid-boot/on-disk layer.
- **Invariant:** semver order == schema-capability order across release branches.
- The node-side **refuse-to-start on any unmodeled key** (in the seam MVP) is the non-cuttable backstop for sidecar↔seid skew and is independent of this scheme.

## Modes

Keep the four — `validator / full / seed / archive`. Modes own **static, role-shaped defaults at generate time only**; **nothing at runtime**. Per-node identity (`moniker`, `persistent_peers`, `external_address`, keys) is **never** a mode default — guard test: `Validate()` fails CI if an identity key appears in any mode's defaults. `DefaultForMode(mode)` stays pure. **The controller already maps its CRD node types to strict library modes** (`fullNode→ModeFull`, `replayer→ModeFull`, `archive→ModeArchive`, `validator→ModeValidator`; `internal/planner/*.go`), so the `full`/`fullNode` enum mismatch does **not** break today — but the translation lives controller-side and must stay guard-tested. `ModeSeed` is never produced by the controller. (Taxonomy → [Appendix A](#appendix-a--modes-beyond-the-core).)

## MVP — the first implementation slice (two coordinated PRs)

The slice spans **sei-chain** (the gated seam + fidelity test + collision audit) and **sei-config** (the `KNOWN_UNMAPPED`/refuse-on-unmodeled enforcement) — two coordinated PRs, not one. The versioning rewrite above is **not** part of it.

**Value:** *a real `seid` home dir resolves through the library and produces a node that behaves identically to the legacy path, behind an off-by-default flag* — which de-risks everything downstream. **In:**

- gate + seam (both channels);
- **fidelity test (non-self-referential):** diff against an external key inventory of a sanitized **real** `config.toml`/`app.toml` — **the rendered `initAppConfig()` template, not the Go mapstructure tags** (the state-store sub-config renders `ss-`-prefixed keys whose mapstructure tags are bare, so comparing the wrong layer yields false corruption reports) — and **not** `WriteConfigToDir`→`ReadConfigFromDir`, which shares the lossy mapper and structurally cannot catch a gap. Because the Cosmos/Sei sections are read lazily via `appOpts.Get("section.key")` with no intermediate struct, a mis-spelled key silently yields the default rather than an error — so the rendered-template diff is the only real guard;
- **close the legacy-coverage gap the test fails on** *(non-exhaustive list; the fail-closed guard below is the real coverage guarantee).* Known today: missing `[admin_server]`, `[rosetta]`, evm `trace_bake_*` + `enable_parallelized_block_trace`, `[state-commit.flatkv]`, `evm-ss-split`/`evm-ss-separate-dbs`, `sc-historical-proof-*`, `sc-snapshot-write-rate-mbps`, `sc-keys-to-migrate-per-block`; and **tag corruptions** — the `eth_block_test` *section name is correct*, but its inner field tags `eth_block_test_*` must become `eth_blocktest_*`; and `ss-evm-db-directory` must become `evm-ss-db-directory`. *Cut option:* a fail-closed guard that refuses to start on any unmodeled key in the operator's file, deferring the field additions (un-defer immediately for archive/RPC fleets, where these keys are present from boot);
- **`KNOWN_UNMAPPED_FIELDS` enforced, not just documented:** mark genesis-sourced/unmapped keys (starting with `chain.chain_id`) **non-settable** — enforce in the shared **`ApplyOverrides`** (so both `ResolveIntent` and `ResolveIncrementalIntent`/day-2 are covered) and surface it in `ValidateIntent` as a dry-run `SeverityError`. Else a controller `spec.Overrides` entry returns `Valid:true` and the value silently vanishes on write;
- **node refuses-to-start on any unmodeled key** (the backstop for sidecar↔seid skew; needs no version stamp);
- the refined collision audit; replicate the `SEI_LOG_LEVEL` extrapolation (`util.go:187-217`) — resolves OQ1.

**Done:** legacy path provably unchanged with the flag unset; gated path starts identically and refuses-to-start on `Validate` errors *and* on unmodeled keys; the fidelity test is green against a real fixture; `make ci` green. (Version-based refuse-to-start is **not** in this slice — see the versioning slice + OQ2.)

## Deferred (un-defer trigger)

- **`seid config …` CLI** → once the seam is proven on a non-prod node. Deterministic exit-code scheme (0 = clean/no-op; nonzero = validation-fail/migration-aborted; distinct code for refuse-on-newer) for initContainer/Job use.
- **Day-2 `ConfigPatchTask` under the library** → when unvalidated raw-TOML merges on running nodes cause an incident or block a config the registry should own.
- **Unified `sei.toml` (Phase 3)** → after the two-file round-trip is trusted on ≥3 real fixtures for a release cycle.
- **Next migration function** → when a shape change advances `CurrentSchemaVersion` past `v6.5.0` (the baseline→`v6.5.0` entry already ships).
- **K8s render-at-init + secret-field enforcement** → before any env uses ConfigMap delivery (until then, secret deny-list documented, not enforced).
- **Mode/CRD alignment** ([Appendix A](#appendix-a--modes-beyond-the-core)) → when generating dedicated `replayer` config is required.
- **Controller/sidecar consolidation onto `seid`** → collapses the three sei-config pins to one; the real fix for write-path skew. Until then the library exposes `ConfigIntent` for the sidecar's use.
- **Continuous canonical materialization + config-introspection endpoint** ([Appendix C](#appendix-c--continuous-canonical-materialization-future), [Appendix D](#appendix-d--config-introspection-endpoint-admin-surface-future)) → builds on the seam to make the rollout observable fleet-wide; un-defer once the seam MVP proves out on a non-prod node.

## Open questions

1. ~~Replicate `SEI_LOG_LEVEL` extrapolation, or accept a delta?~~ → **Resolved: replicate.** Cheap and deterministic; accepting a delta violates the "behaves identically" MVP property and silently no-ops module-level debug toggles (`SEI_LOG_LEVEL=consensus:debug,*:info`) during exactly the incident where you'd flip the gate.
2. Where does on-disk `schema_version` live so **both** the gated manager and the legacy loader can read it — a managed header in `app.toml`, or only the future `sei.toml`? **This is a prerequisite for the versioning slice** (version-based refuse-to-start can't ship until the stamp location is decided) — but *not* for the seam MVP, whose refuse-to-start keys on unmodeled fields, not on a version.
3. Mechanism for pinning the `seictl` sidecar image to the seid image until consolidation lands. Note this is a **per-node vs platform-wide** surface: `EffectiveSidecarImage` falls back to a platform-wide image when `Spec.Sidecar.Image` is unset, so "pin sidecar to seid" is not a single per-node field flip.
4. Final `seid config …` naming/flags.

## Cross-repo coordination

Four repos move together: **sei-config** (library contract — version stamping, registry/`KNOWN_UNMAPPED` enforcement, the semver-validated comparator), **sei-chain** (the gated seam + non-self-referential fidelity test + refined collision audit + user CLI), **seictl** (the sidecar that resolves/writes — *its* pin determines on-disk fidelity), **sei-k8s-controller** (intent; *will* derive image-tag→`TargetVersion` — see versioning slice). The seam PR must not merge until the fidelity test is green and the collision audit passes. Bumping `CurrentSchemaVersion` is a cross-repo event: the sidecar and seid pins must move together, or the writer and reader disagree on shape.

---

## Appendix A — modes beyond the core

- The deployed `SeiNode` CRD union is `validator / fullNode / archive / replayer` — **no `seed`**, and **`fullNode`** (not `full`). The library ships `validator / full / seed / archive`.
- **The controller already translates** CRD → library mode (`fullNode→ModeFull`, `replayer→ModeFull`, `archive→ModeArchive`, `validator→ModeValidator`; `internal/planner/*.go`). A raw CRD string never reaches the library, so the `full`/`fullNode` mismatch is **not a live break**. Note: `replayer`/`fullNode` role identity is *erased only at the `Mode` field* (both → `ModeFull`); their **override sets intentionally differ** (replayer carries mandatory snapshot/peers + a `ResultExport`), so the guard test should assert the `Mode` mapping, not treat the roles as fully indistinguishable. `seed` has no CRD target.
- Aligning the library enum — add dedicated `replayer`, reconcile `seed`, settle `full` vs `fullNode` — is a one-way door only on the **`generate --mode` CLI surface**. The migration registry keys on version, not mode strings, so a future migration can rewrite `cfg.Mode` in place. **Deferred; un-defer when generating dedicated `replayer` config is required.** Keep a guard test asserting every CRD value maps to a valid `NodeMode` or is explicitly rejected.

## Appendix B — `seid config` cobra↔urfave integration

- This is the **user-facing** CLI in the node binary — distinct from the `seictl` *sidecar* that resolves intent in-pod.
- **Delegation:** one `config` cobra command with `DisableFlagParsing: true` (already used at `root.go:176,200`) hands the raw arg tail to the urfave `cli.Command`; urfave owns only the `config` subtree and never sees global cobra flags. Errors propagate via `RunE` to `main.go`'s `os.Exit`; urfave's own exit handler is a no-op.
- **Accepted costs:** go.mod already carries urfave/cli **v2** (load-bearing in `sei-db/…/litt/cli`); v3 adds a second major version — legal, deliberate. Shell completion can't introspect a `DisableFlagParsing` subtree, so `config` subcommands won't autocomplete (deferred).

## Appendix C — Continuous canonical materialization (future)

> Not an MVP deliverable. An extension of the seam that makes the rollout self-proving. (Avoids the word "shadow" — overloaded — in favor of **active vs passive resolution**.)

Produce the canonical config form **on every boot, regardless of which manager is active**, so the canonical artifact is always present, validated, and `schema_version`-stamped *before* `v2` is ever switched on:

- The gate picks the **active** resolver (legacy or `v2`); its result is authoritative and boots the node.
- The other resolver runs **passively** — it resolves and materializes the canonical form (unified `SeiConfig` → legacy TOML, **file-tier**) into `${home}/config/.managed/`, but never feeds the running node.

What it buys:
- **Warm cutover.** Flipping `v2` *promotes an artifact that already exists and has been validated on this node*, not a cold first-run generation.
- **Fleet-wide fidelity/skew canary.** `.managed/` continuously diffs against the authored `config/` (library coverage gaps) and, in the controller fleet, against what the sidecar wrote (sidecar↔seid pin skew) — on real production config, not a CI fixture.
- **Migration dark-launch.** If `.managed/` is **persisted + `schema_version`-stamped across boots** (vs the seam MVP's per-boot-ephemeral treatment), the migration machinery is exercised continuously, so the first real migration at cutover is a no-op rather than a debut. Safe because `.managed/` is **non-authoritative** while `v2` is off — it never touches the operator's authored files, so it does not reawaken the boot-time-rewrite/reversibility concern (which is strictly about authored files).

Invariants (carried from the seam): `.managed/` stays **file-tier** (pre-env/flags, so it remains a valid `v2` read-source and `flag > env > file` precedence holds); the passive path is **non-fatal** (defer/recover — a materialization bug logs and is swallowed, never affects the active boot), **write-isolated** to `.managed/`, and **kill-switchable**, so "legacy path unchanged" stays literally true. This is a real scope addition (an always-on subsystem, leaning on the versioning slice for stamping) — its own decision, deliberately **outside the seam MVP**.

## Appendix D — Config-introspection endpoint (admin surface, future)

> Not an MVP deliverable. The observability capstone of Appendix C.

The config-introspection API **lives on the admin endpoint** — the existing loopback-only admin gRPC server (`[admin_server]`, `admin_address` loopback). That is the right Tier‑1 home: it already runs in-process with access to both the active Viper state and the passive `.managed/` resolution, it is privileged and node-local **by construction** (never the public Tendermint RPC), so the secret-bearing raw view has no network surface to begin with. It returns the resolved configuration in the canonical unified `SeiConfig` form + `schema_version`, and supports comparing the managers — a `manager=v2` / `manager=legacy` selector and a diff of `{key, legacy_value, v2_value, source: default|file|env|flag}` (per-key **provenance** is the thing raw TOML can't give).

**Allowlist / default-deny, enforced at the admin endpoint.** Resolved config is secret-bearing (priv-validator/TLS key paths, `tx_index.psql_conn` credentials, node key, admin address, peer topology). A field may cross the node boundary **only if explicitly allowlisted** — exposing a value is an opt-in act, enforced server-side at the source, so nothing sensitive leaks by default or by omission.

**Fleet visibility (Tier‑2):** because the admin endpoint is loopback-only, the network/fleet view is the sidecar's job — it calls the local admin endpoint in-pod and relays only the **allowlisted projection** outward over its authenticated control-plane channel. The control plane then sweeps sidecars and asks "show every node where `v2 ≠ legacy`," turning the default-flip into a data-driven decision instead of a leap. The admin endpoint owns the API and the allowlist; the sidecar is a thin authenticated relay — consistent with the two-tier model (Tier‑1 seid primitive, Tier‑2 sidecar aggregator) from the [upgrade-shutdown design](../upgrade-shutdown-contract/DESIGN.md). **Not an MVP deliverable.**
