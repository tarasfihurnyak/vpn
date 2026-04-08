-- +migrate Up

CREATE TABLE allowed_ips (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cidr       CIDR        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, cidr)
);

-- +migrate Down

DROP TABLE allowed_ips;
