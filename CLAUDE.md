# wgRift

WireGuard VPN management platform with CLI, REST API, and WASM web UI.

## Build & Development

```bash
make build       # Build CLI binary → bin/wgrift
make wasm        # Compile WASM UI → ui/web/wgrift.wasm
make serve-web   # Dev WASM server on :8080
make serve       # Full local dev (wasm + serve command)
make dist        # Cross-compile linux/amd64 → dist/wgrift
make test        # Run tests (internal/...)
make lint        # golangci-lint
```

Module: `github.com/drudge/wgrift`
Go version: 1.25.5

## Architecture

```
cmd/wgrift/          CLI (Cobra) — interface, peer, status, serve, adopt, version
internal/
  auth/              Session-based auth, bcrypt passwords, CSRF
  config/            YAML config loader with env var support
  confgen/           WireGuard .conf generation & parsing
  models/            Data structs: Interface, Peer, User, Session, ConnectionLog
  server/            HTTP API, middleware, SPA static handler, connection poller
  store/             SQLite via modernc.org/sqlite, migration system
  wg/                WireGuard kernel control via wgctrl + netlink
ui/
  embed.go           go:embed for web assets (index.html, wgrift.wasm, wasm_exec.js)
  web/               WASM UI source (Loom framework)
deploy/              systemd service, config template, installer, Proxmox LXC script
```

### Web Assets Are Embedded

The binary embeds `ui/web/index.html`, `ui/web/wgrift.wasm`, and `ui/web/wasm_exec.js` via `ui/embed.go`. Single-binary deploy — no separate web directory needed.

The SPA static handler (`internal/server/static.go`) serves embedded files and falls back to `index.html` for client-side routes.

### Server vs Client AllowedIPs

These are fundamentally different concepts:
- **Server-side AllowedIPs** (`peers.allowed_ips`): Cryptokey routing — what source IPs to accept FROM the peer
- **Client-side AllowedIPs** (`peers.client_allowed_ips`): What destinations the client should route THROUGH the tunnel

They are stored as separate DB columns and rendered as separate fields in the UI ("Server Allowed IPs" and "Client Allowed IPs").

### DNS Is Client-Only

DNS is a client-side WireGuard directive. Including it in server configs causes `wg-quick` to invoke `resolvconf`, which may not be installed. `GenerateServerConf` intentionally omits DNS.

### Config Comments Break UniFi

UniFi WireGuard clients reject config files that contain comment lines (`# Name`). `GeneratePeerConf` must not emit comments.

## Database

SQLite at `/var/lib/wgrift/wgrift.db`. Migrations in `internal/store/migrations/` are numbered SQL files applied automatically on startup.

When adding a column: create a new migration file (next number), add the column to all relevant SQL queries in `store.go` (INSERT, SELECT, UPDATE) and scan functions.

## Deployment

Target: LXC container 100 on `blue.adk.network`

```bash
make dist
ssh root@blue.adk.network "pct exec 100 -- systemctl stop wgrift"
scp dist/wgrift root@blue.adk.network:/tmp/wgrift
ssh root@blue.adk.network "pct push 100 /tmp/wgrift /usr/local/bin/wgrift && \
  pct exec 100 -- chmod +x /usr/local/bin/wgrift && \
  pct exec 100 -- systemctl start wgrift"
```

Must stop the service before overwriting the binary ("Text file busy").
Binary goes to `/usr/local/bin/wgrift`. Config at `/etc/wgrift/config.yaml`.

## Loom WASM Framework — Critical Gotchas

The web UI uses `github.com/loom-go/loom` and `github.com/loom-go/web`. These are critical patterns learned from debugging:

### Never Use Show for Structural DOM Changes

`Show` has DOM cleanup bugs — nodes aren't properly removed when the condition becomes false, causing ghost elements or broken re-renders. **Always use `Bind` instead.**

### Bind Must Have Identical DOM Structure in Both States

When using `Bind` to toggle visibility, both branches must produce the **same DOM tree structure** (same element types, same nesting). Toggle visibility using CSS classes (`hidden` vs visible), not by adding/removing DOM nodes.

**Bad** (different structure breaks re-render):
```go
Bind(func() loom.Node {
    if visible() {
        return Div(Icon("check", 14), Span(Text("Done")))
    }
    return Div(Icon("copy", 14), Span(Text("Copy")))
})
```

**Good** (identical structure, swap content via innerHTML/text):
```go
Bind(func() loom.Node {
    svg := icons["copy"](14)
    label := "Copy"
    cls := "border-gray-300"
    if visible() {
        svg = icons["check"](14)
        label = "Done"
        cls = "border-green-300"
    }
    return Button(
        Apply(Attr{"class": cls}),
        Span(Apply(innerHTML(svg))),
        Span(Text(label)),
    )
})
```

### Never Use dispatch()

Loom signals are the reactive primitive. There is no `dispatch()` function.

### Icon Rendering in Bind

`Icon(name, size)` generates different DOM trees for different icon names. Inside a `Bind`, use `Span(Apply(innerHTML(icons[name](size))))` instead to keep DOM structure stable.

### Polling Pattern

For auto-refreshing views, use `js.Global().Call("setInterval", ...)` and store the interval ID in a package-level `js.Value`. Clear it in `stopPolling()` (called by the router on route change).

```go
var myPollInterval js.Value

// In view setup:
myPollInterval = js.Global().Call("setInterval", js.FuncOf(func(this js.Value, args []js.Value) any {
    loadData()
    return nil
}), 5000)
```

Register it in `router.go`'s `stopPolling()`.

## Testing

```bash
make test                           # All tests
go test ./internal/confgen/...      # Config parser tests
```

## API Authentication

- Session cookie: `wgrift_session`
- CSRF token required on POST/PUT/DELETE (sent in response body, submitted via header)
- Roles: `admin` (full), `viewer` (read-only)
- First-run setup creates initial admin via `/api/v1/setup`
