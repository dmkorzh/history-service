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
    kafka_broker_list = 'localhost:60246',
    kafka_topic_list = 'parsed-json',
    kafka_group_name = 'clickhouse',
    kafka_format = 'JSONEachRow',
    kafka_num_consumers = 1,
    kafka_flush_interval_ms = 7500,
    kafka_handle_error_mode = 'stream',
    input_format_skip_unknown_fields = 1;

CREATE TABLE IF NOT EXISTS history.raw_failed_queue
(
    message String
) ENGINE = Kafka SETTINGS kafka_broker_list = 'localhost:60246',
    kafka_topic_list = 'failed',
    kafka_group_name = 'clickhouse',
    kafka_format = 'RawBLOB',
    kafka_num_consumers = 1,
    kafka_flush_interval_ms = 7500;

CREATE MATERIALIZED VIEW IF NOT EXISTS history.calls_consumer TO history.calls AS
SELECT *                EXCEPT start, parseDateTime64BestEffortOrZero(start) as start,
       _timestamp_ms as tsEnqueued
FROM history.calls_queue
WHERE _error = '';

CREATE MATERIALIZED VIEW IF NOT EXISTS history.calls_broken_consumer TO history.calls_broken AS
SELECT _raw_message  as message,
       _error        as error,
       _timestamp_ms as tsEnqueued
FROM history.calls_queue
WHERE _error != '';

CREATE MATERIALIZED VIEW IF NOT EXISTS history.raw_failed_consumer TO history.raw_failed AS
SELECT _timestamp_ms  AS tsProcessed,
       message        as cdr,
       _headers.name  as `header.key`,
       _headers.value as `header.value`
FROM history.raw_failed_queue;