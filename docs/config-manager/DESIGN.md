# Configuration manager — design

> The deliverable in *this* PR is this design only. Implementation lands as separate PRs in three repos (sei-config / sei-chain / seictl), sketched here so reviewers can sanity-check the shape. Phase vocabulary is from [CLAUDE.md](../../CLAUDE.md): Phase 2 = today's two-file layout; Phase 3 = unified `sei.toml`.

## Background

A `seid` node's config is spread across `config.toml` (Tendermint), `app.toml` (Cosmos SDK + Sei custom sections: `state-commit`, `state-store`, `evm`, `giga_executor`, `wasm`, `admin_server`, …), `client.toml`, cobra flags, and `SEID_*`/`SEI_*` env vars resolved by Viper. It loads in `seid`'s `PersistentPreRunE` (`cmd/seid/cmd/root.go:79-104` → `InterceptConfigsPreRunHandler` → `interceptConfigs` in the vendored `sei-cosmos` fork).

The **sei-config library already exists and is the asset**: unified `SeiConfig`, `DefaultForMode()`, `Validate()`, a reflection registry mapping dotted keys → `SEI_*` env vars → destination file, `SEI_*`/`SEID_*` resolution, an empty versioned `MigrationRegistry` (`CurrentVersion = 1`), and atomic round-trip IO onto the two legacy files. **Nothing calls it yet.** The risk is entirely at the seam into `seid` and in round-trip fidelity against *real* operator config files — not in the library internals.

## Goals

- An **env-var-gated** path in `seid` that resolves config through the sei-config library instead of the legacy loader. Default off; legacy path byte-for-byte unchanged.
- A management CLI (`seictl`) exposing the library's existing capabilities: `doctor` (validate), `generate --mode` (node-type defaults), `migrate` (schema versioning).
- A versioning contract that lets a config be migrated safely across `seid` releases.
- Keep all changes **inside the sei-chain repo proper** — touch zero lines of the `sei-cosmos` fork.

## Non-goals

- One physical config file (Phase 3). Phase 2 maps the unified config across the existing two files; `sei.toml`-on-disk is deferred.
- Owning `seid init`, `client.toml`, secret material, or hot-reload.
- Weaving a CLI framework into `seid` (see decision below).
- Writing actual migration functions — `CurrentVersion` is 1; there is nothing to migrate yet.

## ⚠️ Decision: urfave/cli v3 lives in `seictl`, not in `seid`

The vision names urfave/cli v3 as the single-file-config UX. But `seid` is cobra-rooted, and two arg-parsers in one binary collide (two help systems, two `os.Args` consumers) for no user benefit. All three consulted specialists landed here independently. **Resolution:** `seid` gets only the runtime *seam* (pure cobra/library call, no CLI framework); the urfave/cli v3 surface ships as **`seictl`**, a separate binary importing the library — a consumer CLAUDE.md already names. This honors the urfave/cli v3 requirement without the collision. Reviewers should confirm this split first.

## Architecture — three pieces, three repos

| Piece | Repo | Role |
|---|---|---|
| `seiconfig` library | sei-config | The brain — exists today. Resolution, modes, validate, migrate, legacy IO. |
| Env-gated seam | sei-chain | `PersistentPreRunE` hook that routes config through the library when gated on. |
| `seictl` (urfave/cli v3) | seictl | Operator/CI surface: `config doctor \| generate \| migrate \| show`. |

**The seam contract (load-bearing).** Inject at `root.go:101-103`, after the existing `init` skip (`:97`). When gated on, the new path must produce exactly what the legacy path does, because `start.go`/`newApp` read config through **two** channels off the server context:

1. `serverCtx.Config` — a fully-populated, `SetRoot`-applied, `ValidateBasic`-passing `*tmcfg.Config`.
2. `serverCtx.Viper` — used as `AppOptions`; every Sei section is read via `appOpts.Get("evm.http_port")`-style dotted lookups.

So the gated path **materializes the two legacy files via the library, then hands off to the same Viper read+merge tail** (`sei-cosmos/server/util.go:162-219, 317-323`) and calls `bindFlags` to preserve flag>env>file precedence. **It must not feed `app.New` from the in-memory `SeiConfig` struct** — that silently drops keys the struct doesn't model. Failure modes to test: Viper left unpopulated (silent zero-value misconfig), flag-precedence inversion, `init`-vs-`start` divergence (gated path must tolerate files it didn't author). `client.toml` is handled before the gate and stays out of scope.

## Env-var gate contract

`SEI_CONFIG_MANAGER`, **value-based** (not presence): `v2` → new path; unset/`legacy` → legacy (default); any other value → hard startup error (never silent fallback). Read with raw `os.Getenv` at the top of `PersistentPreRunE`, before Viper init — it is not itself a config field. Value-based keeps it a clean two-way door: flip to `legacy`, restart, zero residue.

**`SEI_` prefix collision (must-fix).** `seid` already claims `SEI_` via `client.Context.WithViper("SEI")`; the library also uses `SEI_`. With the gate on, both resolve the same env vars — fine *only if they agree on destination*. The implementation PR must ship a **collision audit**: diff Viper's `AutomaticEnv` `SEI_*` keys against the library's `buildEnvMap` output; any disagreement is a release blocker. Precedence ladder (low→high): mode default < file < `SEI_*` env < flag; `SEI_*` wins over deprecated `SEID_*` with a stderr warning.

## Versioning & migration

A `schema_version` integer owned by the registry, **decoupled from the seid release version** (bumps only on config *shape* change; releases→schema in a static code table). `doctor` compares it to `seiconfig.CurrentVersion`: newer → **refuse to start**; older → report "migration available." **No auto-migrate on boot** — migration is explicit (`seictl config migrate`), dry-run by default, writes timestamped `.bak` before `--write`, no-ops when current. Auto-migrate plus the registry's no-downgrade rule is a per-pod one-way door that breaks rollback. The MVP seam only **stamps `schema_version` on write**; the `doctor`/refuse-on-newer/migrate behaviors above ride with the (deferred) CLI and the first real migration — there is nothing to migrate while `CurrentVersion` is 1.

## Modes

Keep the prototype's four — `validator / full / seed / archive` — unchanged. Modes own **static, role-shaped defaults at generate time only**: which indexers/APIs/EVM/state-store are on, pruning posture, listen addresses. Modes own **nothing at runtime** (no enforcement, no drift loop — that's the controller's job). Per-node identity (`moniker`, `p2p.persistent_peers`, `p2p.external_address`, keys) comes from operator/controller overrides and **never** from a mode default. Recommended guard test: `Validate()` fails CI if an identity-bearing key appears in any mode's static defaults. `DefaultForMode(mode)` stays a pure function of the mode enum. Note: `seed` produces operator-CLI defaults only — it has no `SeiNode` CRD target until enum alignment, so the four modes are not all controller-round-trippable.

## MVP — the first implementation PR

**One sentence of value:** *a real `seid` home directory resolves through the library and produces a node that behaves identically to the legacy path, behind an off-by-default env flag.* If that holds, everything downstream (CLI, modes, migration) is de-risked.

In: the gate + seam (both channels populated); the collision audit; a **fidelity test against a sanitized real `config.toml`/`app.toml`** asserting every operator-set key `seid` consumes survives read→write (this is the non-negotiable safety property, not synthetic-default round-trips); a documented `KNOWN_UNMAPPED_FIELDS` list (e.g. `ChainID` lives in genesis.json). Done = legacy path provably unchanged with flag unset; gated path starts identically and refuses-to-start on `Validate` errors; `make ci` green.

## Deferred (with un-defer trigger)

- **`seictl` CLI + urfave/cli v3** → once the seam is proven on a non-prod node (thin wrappers over existing `Validate()`/`DefaultForMode()`).
- **Unified `sei.toml` on disk (Phase 3)** → after the two-file round-trip is trusted on ≥3 real node fixtures for a release cycle.
- **Migration functions** → when the first schema-breaking change forces `CurrentVersion` to 2.
- **K8s render-at-init + secret-field enforcement** → before any environment uses ConfigMap-driven delivery (until then, secret deny-list is documented, not enforced).
- **`replayer` mode / dropping `seed`** → the deployed `SeiNode` CRD union is `validator/fullNode/archive/replayer` (no `seed`); enum alignment is a public-contract change deferred per owner decision.
- **sei-k8s-controller / structured JSON output** → second consumer; library already exposes `ConfigIntent`.

## Open questions

1. Replicate `SEI_LOG_LEVEL` extrapolation (`util.go:187-217`) in the gated path, or accept a documented behavior delta?
2. Where does the on-disk `schema_version` live so a legacy-only checkout can be detected — managed header in `app.toml`, or only in the future `sei.toml`?
3. Final `seictl` command naming/casing (`full` vs CRD's `fullNode`).

## Cross-repo coordination

- **This PR (sei-config):** the design. The library contract (version field, any new facade) follows as a sei-config PR.
- **Follow-up sei-chain PR:** the env-gated seam + fidelity test + collision audit.
- **Follow-up seictl PR:** the urfave/cli v3 management CLI.

The seam PR must not merge until the collision audit passes and the fidelity test is green. `seictl` can follow independently once the library version it pins is tagged.
