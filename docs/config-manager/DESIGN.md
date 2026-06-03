# Configuration manager — design

> This PR ships the design only; implementation lands as separate PRs (sei-config + sei-chain). Phase vocabulary from [CLAUDE.md](../../CLAUDE.md): Phase 2 = today's two-file layout; Phase 3 = unified `sei.toml`.

## Background

A `seid` node's config is spread across `config.toml` (Tendermint), `app.toml` (Cosmos + Sei sections: `evm`, `state-store`, `giga_executor`, …), `client.toml`, cobra flags, and `SEID_*`/`SEI_*` env vars resolved by Viper — loaded in `PersistentPreRunE` (`root.go:79-104` → `InterceptConfigsPreRunHandler` → `interceptConfigs`, in the vendored `sei-cosmos` fork). The **sei-config library already exists and is the asset** (unified `SeiConfig`, `DefaultForMode()`, `Validate()`, a key→env→file registry, `SEI_*`/`SEID_*` resolution, an empty `MigrationRegistry` at `CurrentVersion=1`, atomic two-file IO) — **but nothing calls it yet.** The risk is entirely at the seam into `seid` and round-trip fidelity against *real* config files, not in the library.

## Goals

- An **env-var-gated** path in `seid` resolving config through the library instead of the legacy loader; default off, legacy path byte-for-byte unchanged.
- An in-binary `seid config …` group (`doctor` / `generate --mode` / `migrate`) over the library's existing capabilities.
- A versioning contract for migrating config safely across `seid` releases.
- All changes **inside the sei-chain repo** — zero lines of the `sei-cosmos` fork.

## Non-goals

Phase 3 (one physical `sei.toml`); owning `seid init`, `client.toml`, secrets, or hot-reload; writing migration functions (`CurrentVersion=1`, nothing to migrate); folding sei-config into sei-chain *now* (stays a dependency — see *Future: fold-in*); mode extensions (`replayer`, `seed`/CRD — [Appendix A](#appendix-a--modes-beyond-the-core)).

## ⚠️ Decision: in-binary `seid config …` group on urfave/cli v3

The management UX ships **inside the `seid` binary** as a `seid config …` group on urfave/cli v3 — not a separate `seictl`. One binary, one home for config logic, and CLI-vs-node version skew is impossible (the tool *is* the node binary). The one seam-relevant constraint: **`config` must skip `PersistentPreRunE`** — that hook runs for every subcommand, so without a guard `seid config …` would trigger the legacy interception (and under `SEI_CONFIG_MANAGER=v2`, the gated seam *recursively*), mutating files just to inspect them. Extend the `init` skip at `root.go:97` to also short-circuit `config`. Delegation pattern + accepted costs: [Appendix B](#appendix-b--seid-config-cobraurfave-integration).

## Architecture — two repos

| Piece | Repo | Role |
|---|---|---|
| `seiconfig` library | sei-config | The brain (exists). Resolution, modes, validate, migrate, legacy IO. **Imported by** sei-chain. |
| Env-gated seam | sei-chain | `PersistentPreRunE` hook routing config through the library when gated on. |
| `seid config …` CLI | sei-chain | In-binary surface: `doctor \| generate \| migrate \| show`. |

**Future: fold-in.** sei-config may later collapse into the sei-chain tree so seid owns its own config. The dependency arrow is one-way (sei-chain → sei-config), so that's a `git mv` + import change — **seam unchanged**. Reversible; the only discipline is keeping sei-config a clean leaf, which CLAUDE.md already mandates.

**The seam contract (load-bearing).** Inject at `root.go:101-103`, after the `init` skip. When gated on, the new path must produce exactly what the legacy path does, because `start.go`/`newApp` read config through **two** channels: `serverCtx.Config` (a fully-populated, `SetRoot`/`ValidateBasic`-passing `*tmcfg.Config`) **and** `serverCtx.Viper` (used as `AppOptions` — every Sei section read via `appOpts.Get("evm.http_port")` dotted lookups). So the gated path **materializes the two legacy files, then re-enters the same Viper read+merge tail** (`sei-cosmos/server/util.go:162-219, 317-323`) and calls `bindFlags` for flag>env>file precedence. **It must not feed `app.New` from the in-memory struct** — that silently drops unmodeled keys. Test for: Viper left unpopulated, flag-precedence inversion, `init`-vs-`start` divergence. `client.toml` is handled before the gate, out of scope.

## Env-var gate contract

`SEI_CONFIG_MANAGER`, **value-based**: `v2` → new path; unset/`legacy` → legacy (default); anything else → hard startup error (never silent fallback). Read via raw `os.Getenv` atop `PersistentPreRunE`; keeps a clean two-way door. **`SEI_` collision (must-fix):** `seid` already claims `SEI_` via `WithViper("SEI")` and the library uses `SEI_` too — gated on, both resolve the same env vars, fine *only if they agree on destination*. The implementation PR ships a **collision audit** (diff Viper `AutomaticEnv` `SEI_*` keys vs the library's `buildEnvMap`; any disagreement blocks release). Precedence (low→high): mode default < file < `SEI_*` env < flag; `SEI_*` beats deprecated `SEID_*` (stderr warning).

## Versioning & migration

A `schema_version` integer owned by the registry, **decoupled from the seid release** (bumps only on shape change). `doctor` compares it to `CurrentVersion`: newer → refuse to start; older → "migration available." **No auto-migrate on boot** — migration is explicit (`seid config migrate`), dry-run by default, `.bak` before `--write`, no-ops when current; auto-migrate + no-downgrade is a per-pod one-way door that breaks rollback. The MVP seam only **stamps `schema_version` on write**; `doctor`/refuse-on-newer/migrate ride with the (deferred) CLI and the first real migration.

## Modes

Keep the prototype's four — `validator / full / seed / archive`. Modes own **static, role-shaped defaults at generate time only** (which APIs/EVM/state-store are on, pruning, listen addresses); **nothing at runtime**. Per-node identity (`moniker`, `persistent_peers`, `external_address`, keys) comes from operator/controller overrides, **never** a mode default — guard test: `Validate()` fails CI if an identity key appears in any mode's defaults. `DefaultForMode(mode)` stays pure. (Taxonomy nuance → [Appendix A](#appendix-a--modes-beyond-the-core).)

## MVP — the first implementation PR

**Value:** *a real `seid` home dir resolves through the library and produces a node that behaves identically to the legacy path, behind an off-by-default flag* — which de-risks everything downstream. **In:** gate + seam (both channels); the collision audit; a **fidelity test against a sanitized real `config.toml`/`app.toml`** asserting every operator-set key `seid` consumes survives read→write (the non-negotiable safety property); a `KNOWN_UNMAPPED_FIELDS` list (e.g. `ChainID` lives in genesis.json). **Done:** legacy path provably unchanged with flag unset; gated path starts identically and refuses-to-start on `Validate` errors; `make ci` green.

## Deferred (un-defer trigger)

- **`seid config …` CLI** → once the seam is proven on a non-prod node. Thin wrappers over `Validate()`/`DefaultForMode()`; must ship a deterministic exit-code scheme (0 = clean/no-op; nonzero = validation-fail/migration-aborted; distinct code for refuse-on-newer) for initContainer/Job use.
- **Unified `sei.toml` (Phase 3)** → after the two-file round-trip is trusted on ≥3 real fixtures for a release cycle.
- **Migration functions** → when the first breaking change forces `CurrentVersion`→2.
- **K8s render-at-init + secret-field enforcement** → before any env uses ConfigMap delivery (until then, secret deny-list documented, not enforced).
- **Mode/CRD alignment** ([Appendix A](#appendix-a--modes-beyond-the-core)) → when generating `replayer` nodes is required.
- **sei-k8s-controller / JSON output** → second consumer; library already exposes `ConfigIntent`.

## Open questions

1. Replicate `SEI_LOG_LEVEL` extrapolation (`util.go:187-217`) in the gated path, or accept a documented delta?
2. Where does on-disk `schema_version` live for legacy-only checkouts — a managed header in `app.toml`, or only the future `sei.toml`?
3. Final `seid config …` naming/flags.

## Cross-repo coordination

**This PR (sei-config):** the design; any library contract change (e.g. version stamping) follows as a sei-config PR, tagged for sei-chain to pin. **Follow-up sei-chain PR(s):** the env-gated seam + fidelity test + collision audit, then the in-binary CLI. The seam PR must not merge until the collision audit passes and the fidelity test is green.

---

## Appendix A — modes beyond the core

Out of core scope; captured so the analysis isn't lost.

- The deployed `SeiNode` CRD union is `validator / fullNode / archive / replayer` — **no `seed`**, and **`fullNode`** (not `full`). The prototype ships `validator / full / seed / archive`.
- `seed` produces operator-CLI defaults only and has no CRD target; `replayer` (mandatory snapshot + peers) is first-class in the fleet but absent from the prototype.
- Aligning the enum — add `replayer`, reconcile `seed`, settle `full` vs `fullNode` — is a one-way door only on the **`generate --mode` CLI surface** (the public contract). The migration registry keys on integer version, not mode strings, so a `v1→v2` migration function rewrites `cfg.Mode` in-place and absorbs the rename cleanly. **Deferred per owner decision; un-defer when generating `replayer` nodes is required.**

## Appendix B — `seid config` cobra↔urfave integration

- **Delegation:** one `config` cobra command with `DisableFlagParsing: true` (already used at `root.go:176,200`) hands the raw arg tail to the urfave `cli.Command`; urfave owns only the `config` subtree and never sees global cobra flags. Errors propagate via `RunE` to `main.go`'s `os.Exit`; urfave's own exit handler is a no-op.
- **Accepted costs:** go.mod already carries urfave/cli **v2** (load-bearing in `sei-db/…/litt/cli`); v3 adds a second major version — legal, deliberate. Shell completion can't introspect a `DisableFlagParsing` subtree, so `config` subcommands won't autocomplete (deferred).
