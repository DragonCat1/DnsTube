-- DnsTube initial schema（由历史增量迁移合并；新库仅此一条即可）

CREATE TABLE admin_user (
    id SERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE upstream_groups (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE upstream_servers (
    id SERIAL PRIMARY KEY,
    group_id INT NOT NULL REFERENCES upstream_groups(id) ON DELETE CASCADE,
    address TEXT NOT NULL,
    port INT NOT NULL DEFAULT 53,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_upstream_servers_group ON upstream_servers(group_id, sort_order);

CREATE TABLE dns_instances (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    listen_addr TEXT NOT NULL DEFAULT '0.0.0.0',
    listen_port INT NOT NULL,
    paused BOOLEAN NOT NULL DEFAULT false,
    default_upstream_group_id INT REFERENCES upstream_groups(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (listen_addr, listen_port)
);

CREATE TABLE dns_records (
    id SERIAL PRIMARY KEY,
    instance_id INT NOT NULL REFERENCES dns_instances(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    rtype TEXT NOT NULL,
    ttl INT NOT NULL DEFAULT 300,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_dns_records_instance ON dns_records(instance_id);

CREATE TABLE forward_rules (
    id SERIAL PRIMARY KEY,
    instance_id INT NOT NULL REFERENCES dns_instances(id) ON DELETE CASCADE,
    name_pattern TEXT NOT NULL,
    target_group_id INT NOT NULL REFERENCES upstream_groups(id),
    mode TEXT NOT NULL CHECK (mode IN ('sequential', 'parallel')),
    priority INT NOT NULL DEFAULT 100,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    pattern_source TEXT NOT NULL DEFAULT 'inline' CHECK (pattern_source IN ('inline', 'url')),
    pattern_url TEXT,
    pattern_format TEXT NOT NULL DEFAULT 'regex_list' CHECK (pattern_format IN ('regex_list', 'autoproxy', 'autoproxy_base64')),
    pattern_fetched_body TEXT,
    pattern_resolved_entries JSONB,
    pattern_rule_count INT NOT NULL DEFAULT 0,
    pattern_fetched_at TIMESTAMPTZ,
    hit_count BIGINT NOT NULL DEFAULT 0,
    disabled BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX idx_forward_rules_instance_priority ON forward_rules(instance_id, priority);

CREATE TABLE query_logs (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    instance_id INT NOT NULL REFERENCES dns_instances(id) ON DELETE CASCADE,
    client_ip TEXT NOT NULL,
    qname TEXT NOT NULL,
    qtype TEXT NOT NULL,
    response_code TEXT,
    cache_hit BOOLEAN NOT NULL DEFAULT false,
    forwarded BOOLEAN NOT NULL DEFAULT false,
    upstream_addr TEXT,
    upstream_ms INT,
    total_ms INT,
    error_message TEXT,
    result_summary TEXT,
    forward_upstream_group_id INT REFERENCES upstream_groups(id) ON DELETE SET NULL
);

CREATE INDEX idx_query_logs_created ON query_logs(created_at DESC);
CREATE INDEX idx_query_logs_instance ON query_logs(instance_id);
CREATE INDEX idx_query_logs_instance_created ON query_logs(instance_id, created_at DESC);
CREATE INDEX idx_query_logs_forward_upstream_group ON query_logs(forward_upstream_group_id);
CREATE INDEX idx_query_logs_qtype ON query_logs(qtype);
CREATE INDEX idx_query_logs_response_code ON query_logs(response_code);

CREATE TABLE system_settings (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    telegram_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    telegram_bot_token TEXT,
    telegram_chat_id TEXT,
    telegram_notify_levels TEXT[] NOT NULL DEFAULT ARRAY['warn', 'error']::text[]
);

INSERT INTO system_settings (id) VALUES (1);

CREATE TABLE system_logs (
  id BIGSERIAL PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  kind VARCHAR(16) NOT NULL CHECK (kind IN ('system', 'audit')),
  level VARCHAR(16) NOT NULL DEFAULT 'info',
  event VARCHAR(64),
  message TEXT NOT NULL,
  username VARCHAR(255),
  client_ip VARCHAR(128),
  meta JSONB
);

CREATE INDEX idx_system_logs_created_at ON system_logs (created_at DESC);
CREATE INDEX idx_system_logs_kind ON system_logs (kind);
CREATE INDEX idx_system_logs_level ON system_logs (level);
