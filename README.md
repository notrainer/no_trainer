# No Trainer

A Go server with HTML templates that gives authenticated users one interface to
embedded noVNC desktops and other training targets.

## Projects

These are three independent sibling Git repositories:

| Repository | Purpose |
| --- | --- |
| `no_trainer` | Login, target navigation, HTML templates and bundled browser assets |
| `../no_trainer_client` | Fedora/XFCE browser desktop with Nmap, Zenmap and SSH |
| `../no_trainer_services` | Six deliberately vulnerable network services for lab exercises |

## Structure

- `main.go`: configuration loading, Gin routes, template and static asset embedding.
- `config.go`, `config_test.go`: users/targets parsing, credential lookup and regression tests.
- `jwt.go`, `jwt_test.go`: eight-hour HS256 sessions and token validation tests.
- `templates/`: login, index and error pages; shared elements in `partials/`.
- `assets/main.ts`, `assets/main.css`: Flowbite JavaScript and Tailwind 4 styles.
- `build.mjs`: esbuild/PostCSS build and Chokidar directory watcher.
- `dist/`: generated assets embedded by Go; build these before compiling Go.
- `sample.config.yaml`, `env.sample`: configuration examples, not production secrets.
- `Containerfile`: Alpine runtime for a prebuilt Linux executable, running as UID 65532.
- `.goreleaser.yml`, `.gitlab-ci.yml`: Linux amd64/arm64 builds and tagged releases.
- `compose.yaml`: standalone Webtop example; it does not deploy the Go application.

## Toolchain and dependencies

Use Go 1.27.1+, Node.js 22+ and pnpm 12.8.1. Node/pnpm remain the existing asset
toolchain. `go.mod`/`go.sum` and `package.json`/`pnpm-lock.yaml` are the dependency
sources of truth. `pnpm-workspace.yaml` permits esbuild's install script.

The October 2026 update uses Gin 1.12, JWT v5, maintained `go.yaml.in/yaml/v3`,
Tailwind 4.3, Flowbite 4, Chokidar 5 and TypeScript 7. Tailwind's PostCSS plugin
handles imports, prefixing and optimization, replacing the old separate plugins.
Its CSS configuration lives in `assets/main.css`.

## Development

```bash
pnpm install --frozen-lockfile
cp sample.config.yaml config.yaml
# Edit config.yaml: set a random secret, users, passwords and target URLs.
pnpm watch:dev
```

In another terminal:

```bash
go run .
```

Open `http://localhost:8080`. Configuration is read from `config.yaml`, `.env`,
`NO_TRAINER_*` variables, and `--port`/`--secret` flags. `NO_TRAINER_USERS` and
`NO_TRAINER_TARGETS` accept YAML or JSON; global targets are prepended to each user.
Users support a list of user objects or an object keyed by username. Targets
support a list of target objects or an object mapping target names to URLs:

```dotenv
NO_TRAINER_USERS='{"admin":{"password":"admin","targets":{"client1":"http://localhost:3000"}}}'
```

This reproduces the local training credentials; replace them for deployment.
Invalid users/targets or malformed configuration files stop startup with a clear
error, without printing credentials. Exported environment variables override
`.env`; both override `config.yaml`, and explicit command-line flags take priority.
Rejected login credentials return HTTP 401. The gateway authorization endpoint
returns HTTP 403 for desktops that are not assigned to the authenticated user.
`pnpm dev` builds once; `pnpm watch:dev` watches assets and templates, including
file additions and removals. Go embeds templates and assets at compile time, so
restart `go run .` to see changes in the server.

## Build and validation

```bash
pnpm install --frozen-lockfile
pnpm check
pnpm prod
go vet ./...
go test ./...
CGO_ENABLED=0 go build -trimpath -o no_trainer .
```

Build the runtime image for the local architecture (use `amd64` instead of
`arm64` on an x86 host):

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o build/linux/arm64/no_trainer .
podman build --pull=always -f Containerfile --platform linux/arm64 -t localhost/no_trainer:updated build
```

The image build context is `build/`, containing `linux/<architecture>/no_trainer`.
GoReleaser 2.18.2 supplies this layout automatically through `dockers_v2`.
Current release tooling is GoReleaser 2.18.2 and Docker 29.8.2; the application
runtime is Alpine 3.24.2. Certificate installation runs on the build host's
architecture, allowing amd64/arm64 image assembly without target emulation.
Run `goreleaser check` to validate release configuration. Creating a release tag
triggers GitLab publishing and the configured Telegram announcement.

Frontend assets are embedded in the Go binary, so build them before compiling or
testing Go. Release builds use the committed lockfile and run the Go tests.

## Nmap participant access

Each target can provide optional `ssh_command` and `ssh_host_key` fields in
`NO_TRAINER_USERS`. The authenticated home page shows only the current user's
assigned target, SSH instructions, and presentation links. The SSH username is
`abc`; the deployment configures its password to match the participant's portal
password. The gateway authorizes browser desktop access using the portal session; users
log in once.

Mount the course PDFs read-only at `/materials`: `nmap-training.pdf` and
`nmap-day1.pdf` through `nmap-day3.pdf`. The `/materials/:name` route requires
a valid portal session and permits only these filenames. Sessions last eight
hours so instructions and downloads remain available throughout the training day.

## Deployment checklist

1. Keep deployment Compose files on published `image:` references. Record the
   currently deployed tag/digest before pulling an update.
2. Set a random `NO_TRAINER_SECRET` and your own users; the built-in `changeme`
   secret and sample passwords are insecure. Supply configuration via a private
   env file or a read-only `/app/config.yaml` mount readable by UID 65532.
3. Pull the chosen published image, then restart the application. Bind port 8080
   to loopback or a private network and use an HTTPS reverse proxy.
4. Verify `/login` returns HTTP 200, `/` redirects without a session, and a valid
   login loads target navigation plus `/assets/main.css` and `/assets/main.js`.
5. If verification fails, restore the previous image tag/digest and configuration
   and restart. This app has no database migration.

Sessions expire after eight hours. Passwords are currently compared as plaintext,
and the session cookie is not marked Secure; keep this lab application behind
HTTPS and off public networks until its authentication is hardened. The training
services contain intentional weaknesses and require isolated networking.

## One portal login for desktop access

Configure Nginx `auth_request` on each desktop location, using an internal
subrequest to `GET /internal/auth`. Forward the browser's `Cookie` header and the
original request URI in `X-Original-URI`; discard the request body. The endpoint
returns 204 for the logged-in user's assigned target path, 401 for a missing or
invalid session, and 403 for an unassigned path. Paths are decoded and normalized
before checking permissions, including WebSocket and asset requests.

Keep this authorization subrequest internal to Nginx. Redirect desktop 401
responses to `/login`, remove gateway Basic Auth, and continue supplying the
private desktop's native authorization at the proxy. No desktop password is sent
to the browser. Assigned targets must use distinct, non-root paths such as
`https://nmap.b3n.eu/client1/`.

For a release without Telegram announcements, set the pipeline variable
`NO_TRAINER_SKIP_ANNOUNCE=true`; the existing announcement behavior is otherwise
unchanged.
