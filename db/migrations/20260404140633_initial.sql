-- +migrate Up

CREATE TABLE users (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    username   TEXT        NOT NULL UNIQUE,
    email      TEXT        NOT NULL UNIQUE,
    public_key TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE server_config (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    interface_name TEXT        NOT NULL DEFAULT 'wg0',
    public_key     TEXT        NOT NULL,
    listen_port    INT         NOT NULL DEFAULT 51820,
    ip_pool        CIDR        NOT NULL,
    dns            TEXT[]      NOT NULL DEFAULT '{}',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE peers (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    public_key  TEXT        NOT NULL UNIQUE,
    ip_address  INET        NOT NULL UNIQUE,
    enabled     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +migrate Down

DROP TABLE peers;
DROP TABLE users;
DROP TABLE server_config;
