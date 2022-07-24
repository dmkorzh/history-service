package cmd

const (
	KEY_COLLECTOR_ADDR   = "http-listen.call-collector"
	KEY_ACCESS_LIST      = "http-listen.access-list"
	KEY_REST_SERVER_ADDR = "http-listen.rest-server"

	KEY_BROKERS      = "redpanda.brokers"
	KEY_TOPIC        = "redpanda.topics.calls"
	KEY_TOPIC_FAILED = "redpanda.topics.failed"

	KEY_CLICKHOUSE_HOSTS    = "clickhouse.hosts"
	KEY_CLICKHOUSE_DB       = "clickhouse.database"
	KEY_CLICKHOUSE_USER     = "clickhouse.user"
	KEY_CLICKHOUSE_PASSWORD = "clickhouse.password"

	KEY_LOG = "log"
)
