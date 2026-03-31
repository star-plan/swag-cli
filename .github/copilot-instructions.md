# Copilot Instructions for `swag-cli`

## Build and test commands

- Build the CLI from source with `go build ./cmd/swag-cli`
- Run the full test suite with `go test ./...`
- Run a single package test with `go test ./internal/nginx -run TestDefaultSiteEditor_SetHomepage_IsIdempotent -count=1`
- Other focused package tests follow the same pattern, for example `go test ./internal/config -run TestSaveToLoadFromRoundTrip -count=1`

## High-level architecture

`cmd/swag-cli/main.go` is a thin entrypoint that delegates to the Cobra command tree in `internal/cli`. The root command loads persisted config defaults from `internal/config`, exposes them as persistent flags, and falls back to the interactive Survey-based TUI in `internal/tui` when no subcommand is provided.

Most user-visible operations flow through three shared layers:

- `internal/config` is the source of truth for the SWAG filesystem layout. It resolves paths like `config/nginx/proxy-confs` and `config/nginx/site-confs/default`, expands `~`, normalizes empty values back to defaults, and writes config files atomically.
- `internal/docker` wraps the Docker SDK for container discovery, in-container exec, nginx reloads, and full container restarts. It first tries the normal Docker environment, then falls back to a rootless socket at `XDG_RUNTIME_DIR/docker.sock`.
- `internal/nginx` handles filesystem-facing SWAG changes. `Generator` renders new subdomain proxy configs from the embedded template in `templates/templates.go`, `Manager` lists/toggles/deletes existing proxy confs by parsing the generated nginx variables, and `DefaultSiteEditor` edits the `site-confs/default` homepage block in place.

`internal/swagexport` is a separate export pipeline for packaging a SWAG config directory into a zip. It builds a filtered file plan, applies profile-specific include rules, skips logs/symlinks/sample configs, and always writes a `swag-cli-manifest.json` into the archive.

## Key conventions

- Reuse `internal/config.Config` path helpers instead of hardcoding SWAG paths. Several features depend on compatibility logic there, including fallback from `config/nginx/site-confs/default` to `config/nginx/site-conf/default`.
- Preserve the nginx template variables `set $upstream_app`, `set $upstream_port`, and `set $upstream_proto` when changing proxy generation. `internal/nginx.Manager` relies on those lines to infer the target container/IP and port while listing existing sites.
- Homepage edits are designed to be safe and repeatable: `DefaultSiteEditor` rewrites only the `location /` block inside the 443 `default_server`, keeps runs idempotent, migrates legacy backups, and stores backups under a sibling `.bak` directory.
- File writes are intentionally atomic across the repo. Follow the existing `SaveTo`, `ExportTo`, and `writeFileAtomic` patterns rather than overwriting files directly.
- CLI command handlers typically own user-facing output and termination. In `internal/cli`, the common pattern is colored status/error messages via `fatih/color` plus `os.Exit(1)` for fatal command errors, rather than bubbling errors up through Cobra.
- The TUI and CLI share the same underlying packages. If behavior changes in `internal/nginx`, `internal/docker`, or `internal/config`, check both `internal/cli` and `internal/tui` call sites so the non-interactive and interactive flows stay consistent.
- Export behavior is intentionally conservative. `internal/swagexport` excludes `config/log/**`, ignores symlinks, omits `sample`/`example` proxy configs, and only includes `dns-conf`, `keys`, and `etc/letsencrypt` for `--profile=full --include-secrets`.
- Versioning is injected at release build time via `-ldflags "-X swag-cli/internal/cli.Version=..."` in `scripts/build_push.py`; local builds default to `dev`.
