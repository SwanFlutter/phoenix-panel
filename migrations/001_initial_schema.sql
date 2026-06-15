-- ============================================================
-- PHOENIX PANEL — Initial schema (PostgreSQL dialect)
--
-- This file documents the canonical schema. At runtime the Go backend
-- applies the equivalent schema via GORM AutoMigrate (works on both SQLite
-- and PostgreSQL). Keep this file in sync with internal/models/*.go.
-- ============================================================

BEGIN;

-- ---- admins -------------------------------------------------
CREATE TABLE IF NOT EXISTS admins (
    id            BIGSERIAL PRIMARY KEY,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ,
    username      VARCHAR(64)  NOT NULL,
    password_hash TEXT         NOT NULL,
    role          VARCHAR(16)  NOT NULL DEFAULT 'admin',  -- sudo | admin
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    last_login_ip VARCHAR(64)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_admins_username ON admins(username) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_admins_deleted_at ON admins(deleted_at);

-- ---- settings -----------------------------------------------
CREATE TABLE IF NOT EXISTS settings (
    id         BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    key        VARCHAR(128) NOT NULL,
    value      TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_settings_key ON settings(key) WHERE deleted_at IS NULL;

-- ---- nodes --------------------------------------------------
CREATE TABLE IF NOT EXISTS nodes (
    id           BIGSERIAL PRIMARY KEY,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ,
    name         VARCHAR(128) NOT NULL,
    address      VARCHAR(255) NOT NULL,
    api_host     VARCHAR(255),
    api_port     INTEGER,
    core         VARCHAR(16)  NOT NULL DEFAULT 'xray',     -- xray | sing-box
    status       VARCHAR(16)  NOT NULL DEFAULT 'unknown',
    is_local     BOOLEAN      NOT NULL DEFAULT FALSE,
    is_active    BOOLEAN      NOT NULL DEFAULT TRUE,
    last_seen_at TIMESTAMPTZ,
    xray_version VARCHAR(32)
);
CREATE INDEX IF NOT EXISTS idx_nodes_deleted_at ON nodes(deleted_at);

-- ---- inbounds -----------------------------------------------
CREATE TABLE IF NOT EXISTS inbounds (
    id               BIGSERIAL PRIMARY KEY,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ,
    node_id          BIGINT      NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    tag              VARCHAR(128) NOT NULL,
    protocol         VARCHAR(32)  NOT NULL,                  -- vless|vmess|trojan|shadowsocks|hysteria2|tuic
    listen           VARCHAR(64)  DEFAULT '0.0.0.0',
    port             INTEGER      NOT NULL,
    network          VARCHAR(32)  DEFAULT 'tcp',
    security         VARCHAR(32)  DEFAULT 'none',            -- none|tls|reality
    sni              VARCHAR(255),
    host             VARCHAR(255),
    path             VARCHAR(255),
    flow             VARCHAR(64),
    fingerprint      VARCHAR(32),
    reality_settings TEXT,                                   -- JSON
    extra            TEXT,                                   -- JSON
    is_active        BOOLEAN     NOT NULL DEFAULT TRUE
);
CREATE INDEX IF NOT EXISTS idx_inbounds_node_id ON inbounds(node_id);
CREATE INDEX IF NOT EXISTS idx_inbounds_deleted_at ON inbounds(deleted_at);

-- ---- users --------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id                     BIGSERIAL PRIMARY KEY,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at             TIMESTAMPTZ,
    username               VARCHAR(64) NOT NULL,
    status                 VARCHAR(16) NOT NULL DEFAULT 'active',
    uuid                   VARCHAR(64) NOT NULL,
    trojan_password        VARCHAR(64) NOT NULL,
    ss_password            VARCHAR(64) NOT NULL,
    ss_method              VARCHAR(48) NOT NULL DEFAULT 'chacha20-ietf-poly1305',
    sub_token              VARCHAR(64) NOT NULL,
    data_limit             BIGINT      NOT NULL DEFAULT 0,   -- bytes, 0 = unlimited
    used_up                BIGINT      NOT NULL DEFAULT 0,
    used_down              BIGINT      NOT NULL DEFAULT 0,
    data_strategy          VARCHAR(16) NOT NULL DEFAULT 'no_reset',
    last_reset_at          TIMESTAMPTZ,
    expire_at              TIMESTAMPTZ,                       -- NULL = never
    on_hold_expire_seconds BIGINT,
    note                   VARCHAR(500),
    sub_last_at            TIMESTAMPTZ,
    online_at              TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username  ON users(username)  WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_uuid      ON users(uuid)      WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_sub_token ON users(sub_token) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);

-- ---- user_inbounds (join) -----------------------------------
CREATE TABLE IF NOT EXISTS user_inbounds (
    user_id    BIGINT NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    inbound_id BIGINT NOT NULL REFERENCES inbounds(id) ON DELETE CASCADE,
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    PRIMARY KEY (user_id, inbound_id)
);

-- ---- traffic_logs -------------------------------------------
CREATE TABLE IF NOT EXISTS traffic_logs (
    id         BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    node_id    BIGINT,
    up         BIGINT NOT NULL,
    down       BIGINT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_traffic_logs_user_id ON traffic_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_traffic_logs_node_id ON traffic_logs(node_id);

-- ---- audit_logs ---------------------------------------------
CREATE TABLE IF NOT EXISTS audit_logs (
    id         BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    actor_type VARCHAR(16),
    actor_id   BIGINT,
    actor_name VARCHAR(64),
    action     VARCHAR(64) NOT NULL,
    target     VARCHAR(128),
    ip         VARCHAR(64),
    user_agent VARCHAR(255),
    success    BOOLEAN NOT NULL DEFAULT TRUE,
    detail     TEXT
);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor  ON audit_logs(actor_type, actor_id);

COMMIT;
