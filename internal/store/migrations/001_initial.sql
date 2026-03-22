CREATE TABLE IF NOT EXISTS _migrations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS interfaces (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL CHECK(type IN ('site-to-site', 'client-access')),
    listen_port INTEGER NOT NULL,
    private_key_encrypted TEXT NOT NULL,
    address TEXT NOT NULL,
    dns TEXT NOT NULL DEFAULT '',
    mtu INTEGER NOT NULL DEFAULT 1420,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS peers (
    id TEXT PRIMARY KEY,
    interface_id TEXT NOT NULL REFERENCES interfaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    public_key TEXT NOT NULL,
    private_key_encrypted TEXT NOT NULL DEFAULT '',
    preshared_key_encrypted TEXT,
    allowed_ips TEXT NOT NULL,
    endpoint TEXT,
    persistent_keepalive INTEGER NOT NULL DEFAULT 0,
    enabled INTEGER NOT NULL DEFAULT 1,
    expires_at DATETIME,
    last_handshake DATETIME,
    transfer_rx INTEGER NOT NULL DEFAULT 0,
    transfer_tx INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_peers_interface_id ON peers(interface_id);
CREATE UNIQUE INDEX idx_peers_public_key ON peers(public_key);
