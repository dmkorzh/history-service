CREATE DATABASE IF NOT EXISTS history;

CREATE TABLE IF NOT EXISTS history.calls
(
    start          DateTime64(3, 'UTC'),
    departmentID   LowCardinality(String),
    departmentName String,
    id             UUID,
    callType       Enum8('incoming' = 0, 'outgoing' = 1),
    answered       UInt8,
    clientNumber   UInt64,
    employeeNumber UInt64,
    employeeName   String,
    waiting        UInt32,
    duration       UInt32,
    tsEnqueued     DateTime64(3, 'UTC') CODEC (DoubleDelta) TTL toDateTime(tsEnqueued) + INTERVAL 3 MONTH
) ENGINE = ReplacingMergeTree
      PARTITION BY toYYYYMM(start)
      ORDER BY (departmentID, start, id)
      PRIMARY KEY (departmentID, start)
      TTL toDateTime(start) + INTERVAL 3 YEAR;