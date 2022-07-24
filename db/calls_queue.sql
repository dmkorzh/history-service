CREATE TABLE IF NOT EXISTS history.calls_queue
(
    start          String,
    departmentID   String,
    departmentName String,
    id             UUID,
    callType Enum8('incoming' = 0, 'outgoing' = 1),
    answered       UInt8,
    clientNumber   UInt64,
    employeeNumber UInt64,
    employeeName   String,
    waiting        UInt32,
    duration       UInt32
) ENGINE = Kafka SETTINGS
    kafka_broker_list = 'localhost:9092',
    kafka_topic_list = 'parsed-json',
    kafka_group_name = 'clickhouse',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 1,
    kafka_flush_interval_ms = 7500,
    kafka_handle_error_mode = 'stream',
    input_format_skip_unknown_fields = 1;

CREATE MATERIALIZED VIEW IF NOT EXISTS history.calls_mv TO history.calls AS
SELECT *                EXCEPT start, parseDateTime64BestEffortOrZero(start) as start,
       _timestamp_ms as tsEnqueued
FROM history.calls_queue
WHERE _error = '';