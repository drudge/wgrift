-- General settings key-value store
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- OIDC provider configuration (admin-managed via UI)
CREATE TABLE IF NOT EXISTS oidc_providers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    issuer TEXT NOT NULL,
    client_id TEXT NOT NULL,
    client_secret_encrypted TEXT NOT NULL,
    scopes TEXT NOT NULL DEFAULT 'openid profile email groups',
    auto_discover INTEGER NOT NULL DEFAULT 1,
    admin_claim TEXT NOT NULL DEFAULT '',
    admin_value TEXT NOT NULL DEFAULT '',
    default_role TEXT NOT NULL DEFAULT 'viewer',
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- OIDC user identity link
ALTER TABLE users ADD COLUMN oidc_provider TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN oidc_subject TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX idx_users_oidc_identity
    ON users(oidc_provider, oidc_subject)
    WHERE oidc_provider != '';

-- Ephemeral OIDC state for authorization flow
CREATE TABLE IF NOT EXISTS oidc_states (
    state TEXT PRIMARY KEY,
    provider_id TEXT NOT NULL,
    nonce TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
