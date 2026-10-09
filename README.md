# no_trainer short for No_VNC_Training_Platform

no trainer allows connecting to novnc clients via a unified interface.


## Development

```bash
# Terminal 1
pnpm install --frozen-lockfile
pnpm watch:dev

# Terminal 2
go run .
```

## Build

Use pnpm 8.15.9 (also recorded in `package.json` and the release runner).

```bash
pnpm install --frozen-lockfile
pnpm prod
go test ./...
CGO_ENABLED=0 go build -trimpath -o no_trainer .
docker build -f Containerfile -t no_trainer .
```

Frontend assets are embedded in the Go binary, so build them before compiling or
testing Go. Release builds use the committed lockfile and run the Go tests.
A version tag triggers the existing GitLab release and container publishing job.

## Nmap participant access

Each target can provide optional `ssh_command` and `ssh_host_key` fields in
`NO_TRAINER_USERS`. The authenticated home page shows only the current user's
assigned target, SSH instructions, and presentation links. The SSH username is
`abc`; the deployment configures its password to match the participant's portal
password. Browser desktop authentication still passes through the gateway.

Mount the course PDFs read-only at `/materials`: `nmap-training.pdf` and
`nmap-day1.pdf` through `nmap-day3.pdf`. The `/materials/:name` route requires
a valid portal session and permits only these filenames. Sessions last eight
hours so instructions and downloads remain available throughout the training day.

