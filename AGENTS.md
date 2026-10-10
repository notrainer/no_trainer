# Working on No Trainer

## Scope and map

This repository is the Go/SSR control interface. Related independent repositories
are `../no_trainer_client` (desktop/scanner image) and `../no_trainer_services`
(vulnerable lab images); each has its own README, AGENTS.md and Git status.
Inspect all relevant repositories before making a change and preserve existing
local edits. Do not commit, publish, deploy, or message anyone unless authorized.

- `main.go` loads Viper/.env configuration, defines `/`, `/login`, and `/assets`,
  and embeds `templates/*` plus `dist/*`.
- `config.go` parses list/keyed-object users and list/name-to-URL targets;
  `config_test.go` verifies both formats, overrides and invalid configuration; `jwt.go` implements eight-hour HS256
  sessions; `jwt_test.go` checks signatures, expiration, algorithms and user claims.
- `templates/*.html` and `templates/partials/` are Go HTML templates.
- `assets/main.ts` imports Flowbite; `assets/main.css` configures Tailwind 4,
  source scanning, dark mode and the primary color palette.
- `build.mjs` uses esbuild and `@tailwindcss/postcss`; Chokidar 5 watches directories.
- `dist/`, `build/`, `node_modules/`, and `no_trainer` are generated/ignored.
- `sample.config.yaml` and `env.sample` document configuration; never read out or
  commit `.env`, `config.yaml`, secrets, or real user credentials.
- `Containerfile` packages prebuilt `linux/<arch>/no_trainer` from a build context;
  `.goreleaser.yml` uses `dockers_v2`; `.github/workflows/release.yml` publishes
  tagged GitHub releases and GHCR images. `.github/workflows/ci.yml` validates
  pushes and pull requests. Releases do not send announcements.
- `compose.yaml` is a separate Webtop example, not the application deployment.

## Commands and constraints

Keep Go 1.27.1+, Node 22+ and pnpm 12.8.1 for this existing project; avoid an
unrelated package-manager migration. Commit manifests and lockfiles together.
Use `pnpm install --frozen-lockfile`, `pnpm check`, `pnpm prod`, `go vet ./...`,
`go test ./...`, and `CGO_ENABLED=0 go build -trimpath -o no_trainer .`.
Always build assets before Go because they are embedded. `pnpm watch:dev` rebuilds
assets but the Go server must restart to embed them again. Keep CSS sources and
Flowbite integration in `assets/main.css`, not an old Tailwind JS config.

For image verification, cross-compile to `build/linux/<arch>/no_trainer`, then use
`podman build --pull=always -f Containerfile --platform linux/<arch>
-t localhost/no_trainer:updated build`. Validate release config with
`goreleaser check`; never run a real release just to test configuration.

Prefer small reversible changes and focused tests. Run gofmt on Go edits and
update README/deployment instructions when behavior or tooling changes. Verify
login, redirects, static assets and target navigation after dependency upgrades.
Deployment Compose must use published images. Pull, restart, verify readiness,
and retain a previous digest for rollback when deployment is explicitly requested.
Keep the lab private; do not expand exposure. Default/sample credentials are not
production credentials, and current plaintext passwords/cookie settings are
known limitations. Never log configuration containing secrets. `.env` is loaded from the process
working directory; existing exported variables override it. Config parse errors
must stop startup, not silently produce an empty user list. The app returns 401
for rejected credentials; a 403 needs investigation of the proxy/client layer.
