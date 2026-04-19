ALTER TABLE upstream_servers
    DROP COLUMN IF EXISTS tls_server_name,
    DROP COLUMN IF EXISTS path,
    DROP COLUMN IF EXISTS protocol;
