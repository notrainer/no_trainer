# No Trainer

A Go server with HTML templates that gives authenticated users one interface to
embedded noVNC desktops and other training targets.

Source, CI and releases are hosted at [notrainer/no_trainer](https://github.com/notrainer/no_trainer).
Published images are available as `ghcr.io/notrainer/no_trainer:latest` and
version tags such as `ghcr.io/notrainer/no_trainer:1.1.0`, for Linux amd64 and arm64.

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
- `assets/nmap-visualizer.ts`: local XML parsing, filtering and safe report rendering.
- `build.mjs`: esbuild/PostCSS build and Chokidar directory watcher.
- `dist/`: generated assets embedded by Go; build these before compiling Go.
- `sample.config.yaml`, `env.sample`: configuration examples, not production secrets.
- `Containerfile`: Alpine runtime for a prebuilt Linux executable, running as UID 65532.
- `.goreleaser.yml`, `.github/workflows/`: validation, Linux amd64/arm64 builds and tagged releases.
- `compose.yaml`: standalone Webtop example; it does not deploy the Go application.

## Toolchain and dependencies

Use Go 1.27.1+, Node.js 22.22+ and pnpm 12.8.1. Node/pnpm remain the existing asset
toolchain. `go.mod`/`go.sum` and `package.json`/`pnpm-lock.yaml` are the dependency
sources of truth. `pnpm-workspace.yaml` permits esbuild's install script.

The October 2026 update uses Gin 1.12, JWT v5, maintained `go.yaml.in/yaml/v3`,
Tailwind 4.3, Flowbite 4, Chokidar 6 and TypeScript 7. Tailwind's PostCSS plugin
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

For a local XML viewer preview, build the image above with tag
`localhost/no_trainer:xml-preview` and start it on loopback with test credentials:

```bash
NO_TRAINER_SECRET="$(openssl rand -hex 32)" podman run -d \
  --name no-trainer-xml-preview \
  --read-only --cap-drop=all --security-opt=no-new-privileges \
  -p 127.0.0.1:18087:8080 \
  -e NO_TRAINER_SECRET \
  -e NO_TRAINER_NMAP_VISUALIZER=true \
  -e 'NO_TRAINER_USERS={"tester":{"password":"nmap-test","targets":[]}}' \
  --health-cmd='wget -q -O /dev/null http://127.0.0.1:8080/login' \
  --health-interval=5s --health-timeout=3s --health-retries=3 \
  localhost/no_trainer:xml-preview
```

Open `http://127.0.0.1:18087`, log in with `tester` / `nmap-test`, and choose
**Nmap XML ansehen**. This preview contains no desktop targets or course PDFs.
Check it with `podman healthcheck run no-trainer-xml-preview`; stop it with
`podman stop no-trainer-xml-preview` and resume with `podman start no-trainer-xml-preview`.

Current release tooling is GoReleaser 2.18.2, UPX 5.2.1 and Docker Buildx on
GitHub Actions; the application runtime is Alpine 3.24.2. Certificate installation runs on the build host's
architecture, allowing amd64/arm64 image assembly without target emulation.
Run `goreleaser check` to validate release configuration. Pushes to `master` and
pull requests run asset checks, Go vet/tests, a binary build and release configuration
validation. Pushing a new `v*` version tag publishes GitHub release assets and
GHCR images using the workflow's built-in `GITHUB_TOKEN`; no personal release
token or announcement secret is required. Releases do not send announcements.

The GitLab commit history and tags were preserved during migration. Historical
releases `v1.0.0` through `v1.0.3` retain their original downloadable binaries and
checksum files on GitHub; their notes record the original release dates. Tags
`v1.0.4` through `v1.0.6` had no GitLab release assets to migrate. The old GitLab
registry remains available for rollback while deployments move to GHCR.

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

## Standalone Nmap XML viewer

Open [`standalone/nmap-visualizer.html`](standalone/nmap-visualizer.html) directly
in your browser (double-click or `file://`), with no server, login or installation
needed. Tailwind's browser script and generated styles load from jsDelivr
(version 4.3.3), so an internet connection is required. Only the viewer's custom
CSS and application logic are embedded. The inline script is formatted with
readable function and variable names, without minification. You can copy the HTML anywhere.
XML files stay in the browser. Search, port-state
filters, host details and NSE output work like the portal viewer. If the browser
blocks clipboard access for local files, select and copy the NSE text manually.

The HTML is committed as a ready-to-use artifact and is independent of the
portal build and deployment. To refresh it after viewer changes, use the existing
Node/pnpm toolchain:

```bash
pnpm install --frozen-lockfile
node build-standalone.mjs
```

## Deployment checklist

### Optional Nmap XML viewer

Set `NO_TRAINER_NMAP_VISUALIZER=true` in the portal's environment (or
`nmap_visualizer: true` in `config.yaml`) to enable the **Nmap XML ansehen**
sidebar tab for all authenticated participants. It is disabled by default.
Invalid boolean values stop startup. Environment settings override the YAML file.
Restart the portal after changing the setting.

The viewer is a small independent implementation inspired by the
[Nmap XML Visualizer](https://github.com/sumanrox/sumanrox.github.io).
Open an Nmap XML report produced with `-oX` or `-oA` (up to 10 MiB) to see host
addresses/names, host status, OS matches, ports, services, versions and NSE output.
Search filters hosts; the port-state selector filters their displayed ports.
The Tailwind report table separates ports and protocols, and shows status badges
(green for open/up, red for closed/down, amber for filtered or mixed states).
Each NSE script has an expandable preview, a scrollable monospace output and a
copy button. Host-level scripts use the same presentation. Duplicate addresses
and hostnames appear only once in the host header.
Collapsed port summaries and discovery-only scans are supported.
Files are parsed locally in the browser, never uploaded or stored, and are cleared
on reset or page reload. There is no extra top bar, analytics, external CDN or
external report link. The portal's existing desktop navigation remains available.
The narrow icon sidebar stays visible on mobile screens so all tabs are reachable.

For Compose deployments, explicitly pass the switch to the **portal service**;
an entry in the Compose env file alone does not add it to the container:

```yaml
environment:
  NO_TRAINER_NMAP_VISUALIZER: "${NO_TRAINER_NMAP_VISUALIZER:-false}"
```

Use a published image containing the viewer, pull it, and recreate only the
portal service. Verify login, the new sidebar tab, opening/filtering/resetting a
report, and existing desktop navigation. Set the switch to `false` and recreate
the portal to disable the feature; keep the previous image digest for rollback.

### Portal rollout

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
