package config

import (
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App      AppConfig      `yaml:"app"`
	Kafka    KafkaConfig    `yaml:"kafka"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Tracing  TracingConfig  `yaml:"tracing"`
	Schema   SchemaConfig   `yaml:"schema"`
}

type TracingConfig struct {
	Enabled      bool    `yaml:"enabled"`
	CollectorURL string  `yaml:"collector_url"`
	SampleRate   float64 `yaml:"sample_rate"`
}

type SchemaConfig struct {
	SchemasDir string `yaml:"schemas_dir"`
}

type AppConfig struct {
	Name    string `yaml:"name"`
	Env     string `yaml:"env"`
	Port    int    `yaml:"port"`
	Metrics string `yaml:"metrics"`
}

type KafkaConfig struct {
	Brokers       []string       `yaml:"brokers"`
	ConsumerGroup string         `yaml:"consumer_group"`
	Topics        TopicsConfig   `yaml:"topics"`
	Producer      ProducerConfig `yaml:"producer"`
	Consumer      ConsumerConfig `yaml:"consumer"`
}

type TopicsConfig struct {
	TransactionsPending    string `yaml:"transactions_pending"`
	TransactionsAuthorized string `yaml:"transactions_authorized"`
	TransactionsSettled    string `yaml:"transactions_settled"`
	TransactionsFailed     string `yaml:"transactions_failed"`
	TransactionsReversed   string `yaml:"transactions_reversed"`
	FraudEvents            string `yaml:"fraud_events"`
	AuditLog               string `yaml:"audit_log"`
	NotificationsOutbox    string `yaml:"notifications_outbox"`
	NotificationsSent      string `yaml:"notifications_sent"`
	NotificationsFailed    string `yaml:"notifications_failed"`
}

func (t TopicsConfig) Defaults() TopicsConfig {
	if t.TransactionsPending == "" {
		t.TransactionsPending = "transactions.pending"
	}
	if t.TransactionsAuthorized == "" {
		t.TransactionsAuthorized = "transactions.authorized"
	}
	if t.TransactionsSettled == "" {
		t.TransactionsSettled = "transactions.settled"
	}
	if t.TransactionsFailed == "" {
		t.TransactionsFailed = "transactions.failed"
	}
	if t.TransactionsReversed == "" {
		t.TransactionsReversed = "transactions.reversed"
	}
	if t.FraudEvents == "" {
		t.FraudEvents = "fraud.events"
	}
	if t.AuditLog == "" {
		t.AuditLog = "audit.log"
	}
	if t.NotificationsOutbox == "" {
		t.NotificationsOutbox = "notifications.outbox"
	}
	if t.NotificationsSent == "" {
		t.NotificationsSent = "notifications.sent"
	}
	if t.NotificationsFailed == "" {
		t.NotificationsFailed = "notifications.failed"
	}
	return t
}

type ProducerConfig struct {
	Idempotence       bool   `yaml:"idempotence"`
	Acks              string `yaml:"acks"`
	Compression       string `yaml:"compression"`
	LingerMs          int    `yaml:"linger_ms"`
	BatchSize         int    `yaml:"batch_size"`
	BufferMemory      int    `yaml:"buffer_memory"`
	MaxInFlight       int    `yaml:"max_in_flight"`
	Retries           int    `yaml:"retries"`
	RetryBackoffMs    int    `yaml:"retry_backoff_ms"`
	RequestTimeoutMs  int    `yaml:"request_timeout_ms"`
	DeliveryTimeoutMs int    `yaml:"delivery_timeout_ms"`
}

type ConsumerConfig struct {
	AutoOffsetReset        string `yaml:"auto_offset_reset"`
	EnableAutoCommit       bool   `yaml:"enable_auto_commit"`
	IsolationLevel         string `yaml:"isolation_level"`
	MaxPollIntervalMs      int    `yaml:"max_poll_interval_ms"`
	SessionTimeoutMs       int    `yaml:"session_timeout_ms"`
	HeartbeatIntervalMs    int    `yaml:"heartbeat_interval_ms"`
	FetchMinBytes          int    `yaml:"fetch_min_bytes"`
	FetchMaxWaitMs         int    `yaml:"fetch_max_wait_ms"`
	MaxPartitionFetchBytes int    `yaml:"max_partition_fetch_bytes"`
}

type DatabaseConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	User            string `yaml:"user"`
	Password        string `yaml:"password"`
	Name            string `yaml:"name"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime int    `yaml:"conn_max_idle_time"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

func Load(path string) (*Config, error) {
	var cfg Config

	if path == "" {
		path = os.Getenv("CONFIG_PATH")
	}

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
		} else {
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return nil, err
			}
		}
	}

	applyEnvOverrides(&cfg)
	cfg.Kafka.Topics = cfg.Kafka.Topics.Defaults()

	return &cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("APP_NAME"); v != "" {
		cfg.App.Name = v
	}
	if v := os.Getenv("APP_ENV"); v != "" {
		cfg.App.Env = v
	}
	if v := os.Getenv("APP_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.App.Port = n
		}
	}
	if v := os.Getenv("APP_METRICS"); v != "" {
		cfg.App.Metrics = v
	}

	if v := os.Getenv("KAFKA_BROKERS"); v != "" {
		cfg.Kafka.Brokers = splitCSV(v)
	}
	if v := os.Getenv("KAFKA_CONSUMER_GROUP"); v != "" {
		cfg.Kafka.ConsumerGroup = v
	}
	setTopicEnv(&cfg.Kafka.Topics.TransactionsPending, "KAFKA_TOPIC_TRANSACTIONS_PENDING")
	setTopicEnv(&cfg.Kafka.Topics.TransactionsAuthorized, "KAFKA_TOPIC_TRANSACTIONS_AUTHORIZED")
	setTopicEnv(&cfg.Kafka.Topics.TransactionsSettled, "KAFKA_TOPIC_TRANSACTIONS_SETTLED")
	setTopicEnv(&cfg.Kafka.Topics.TransactionsFailed, "KAFKA_TOPIC_TRANSACTIONS_FAILED")
	setTopicEnv(&cfg.Kafka.Topics.TransactionsReversed, "KAFKA_TOPIC_TRANSACTIONS_REVERSED")
	setTopicEnv(&cfg.Kafka.Topics.FraudEvents, "KAFKA_TOPIC_FRAUD_EVENTS")
	setTopicEnv(&cfg.Kafka.Topics.AuditLog, "KAFKA_TOPIC_AUDIT_LOG")
	setTopicEnv(&cfg.Kafka.Topics.NotificationsOutbox, "KAFKA_TOPIC_NOTIFICATIONS_OUTBOX")
	setTopicEnv(&cfg.Kafka.Topics.NotificationsSent, "KAFKA_TOPIC_NOTIFICATIONS_SENT")
	setTopicEnv(&cfg.Kafka.Topics.NotificationsFailed, "KAFKA_TOPIC_NOTIFICATIONS_FAILED")

	if v := os.Getenv("DATABASE_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("DATABASE_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Database.Port = n
		}
	}
	if v := os.Getenv("DATABASE_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := os.Getenv("DATABASE_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("DATABASE_NAME"); v != "" {
		cfg.Database.Name = v
	}
	if v := os.Getenv("DATABASE_MAX_OPEN_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Database.MaxOpenConns = n
		}
	}
	if v := os.Getenv("DATABASE_MAX_IDLE_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Database.MaxIdleConns = n
		}
	}
	if v := os.Getenv("DATABASE_CONN_MAX_LIFETIME_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Database.ConnMaxLifetime = n
		}
	}
	if v := os.Getenv("DATABASE_CONN_MAX_IDLE_TIME_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Database.ConnMaxIdleTime = n
		}
	}

	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := os.Getenv("REDIS_DB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Redis.DB = n
		}
	}
	if v := os.Getenv("REDIS_POOL_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Redis.PoolSize = n
		}
	}

	if v := os.Getenv("OTEL_ENABLED"); v != "" {
		cfg.Tracing.Enabled = strings.ToLower(v) == "true" || v == "1"
	}
	if v := os.Getenv("OTEL_COLLECTOR_URL"); v != "" {
		cfg.Tracing.CollectorURL = v
	}
	if v := os.Getenv("OTEL_SAMPLE_RATE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Tracing.SampleRate = f
		}
	}

	if v := os.Getenv("AVRO_SCHEMAS_DIR"); v != "" {
		cfg.Schema.SchemasDir = v
	}
}

func setTopicEnv(field *string, env string) {
	if v := os.Getenv(env); v != "" {
		*field = v
	}
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
