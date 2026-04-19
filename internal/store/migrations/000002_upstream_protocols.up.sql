-- 为上游服务器引入协议字段，支持 UDP / DoT / DoH 三种传输。
-- protocol 缺省为 'udp' 以保持向后兼容。
ALTER TABLE upstream_servers
    ADD COLUMN protocol TEXT NOT NULL DEFAULT 'udp'
        CHECK (protocol IN ('udp', 'dot', 'doh')),
    ADD COLUMN path TEXT,
    ADD COLUMN tls_server_name TEXT;
