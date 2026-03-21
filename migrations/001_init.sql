-- Initial schema for the observability platform.
-- These are also auto-applied by the Go server on startup via Migrate().

CREATE DATABASE IF NOT EXISTS observability;

CREATE TABLE IF NOT EXISTS observability.logs (
    id          String,
    tenant_id   String,
    timestamp   DateTime64(3),
    level       LowCardinality(String),
    service     LowCardinality(String),
    message     String,
    trace_id    String DEFAULT '',
    span_id     String DEFAULT '',
    attributes  Map(String, String),
    INDEX idx_message message TYPE tokenbf_v1(10240, 3, 0) GRANULARITY 4
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (tenant_id, service, timestamp)
TTL toDateTime(timestamp) + INTERVAL 30 DAY;

CREATE TABLE IF NOT EXISTS observability.spans (
    trace_id       String,
    span_id        String,
    parent_span_id String DEFAULT '',
    tenant_id      String,
    service        LowCardinality(String),
    operation      String,
    start_time     DateTime64(6),
    end_time       DateTime64(6),
    duration_ns    Int64,
    status         UInt8,
    attributes     Map(String, String),
    events         Nested(
        name       String,
        timestamp  DateTime64(6),
        attrs      Map(String, String)
    )
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(start_time)
ORDER BY (tenant_id, service, trace_id, start_time)
TTL toDateTime(start_time) + INTERVAL 7 DAY;

CREATE TABLE IF NOT EXISTS observability.metrics (
    tenant_id  String,
    name       LowCardinality(String),
    value      Float64,
    timestamp  DateTime64(3),
    labels     Map(String, String),
    type       UInt8
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (tenant_id, name, timestamp)
TTL toDateTime(timestamp) + INTERVAL 90 DAY;

CREATE TABLE IF NOT EXISTS observability.alert_rules (
    id         String,
    tenant_id  String,
    name       String,
    query      String,
    type       LowCardinality(String),
    condition  String,
    threshold  Float64,
    duration   Int64,
    channels   Array(String),
    enabled    UInt8,
    created_at DateTime64(3)
) ENGINE = ReplacingMergeTree()
ORDER BY (tenant_id, id);

CREATE TABLE IF NOT EXISTS observability.alerts (
    id          String,
    rule_id     String,
    tenant_id   String,
    status      LowCardinality(String),
    message     String,
    fired_at    DateTime64(3),
    resolved_at Nullable(DateTime64(3))
) ENGINE = MergeTree()
ORDER BY (tenant_id, fired_at);

CREATE TABLE IF NOT EXISTS observability.dashboards (
    id         String,
    tenant_id  String,
    name       String,
    panels     String,
    created_at DateTime64(3),
    updated_at DateTime64(3)
) ENGINE = ReplacingMergeTree(updated_at)
ORDER BY (tenant_id, id);
