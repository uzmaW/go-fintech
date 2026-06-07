package config

import (
	"os"
	"path/filepath"
	"testing"
)

func setEnv(t *testing.T, key, value string) {
	t.Helper()
	t.Setenv(key, value)
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.App.Name != "" {
		t.Errorf("App.Name = %q, want empty", cfg.App.Name)
	}
	if cfg.App.Port != 0 {
		t.Errorf("App.Port = %d, want 0", cfg.App.Port)
	}
	if cfg.Kafka.Brokers != nil {
		t.Errorf("Kafka.Brokers = %v, want nil", cfg.Kafka.Brokers)
	}
	if cfg.Database.Host != "" {
		t.Errorf("Database.Host = %q, want empty", cfg.Database.Host)
	}
	if cfg.Redis.Addr != "" {
		t.Errorf("Redis.Addr = %q, want empty", cfg.Redis.Addr)
	}

	defaults := TopicsConfig{}
	defaults = defaults.Defaults()
	if cfg.Kafka.Topics != defaults {
		t.Errorf("Kafka.Topics not equal to defaults: got %v, want %v", cfg.Kafka.Topics, defaults)
	}
}

func TestLoadFromEnv(t *testing.T) {
	envs := map[string]string{
		"APP_NAME":                         "testapp",
		"APP_ENV":                          "production",
		"APP_PORT":                         "8080",
		"APP_METRICS":                      ":9090",
		"KAFKA_BROKERS":                    "broker1:9092,broker2:9092",
		"KAFKA_CONSUMER_GROUP":             "my-group",
		"KAFKA_TOPIC_TRANSACTIONS_PENDING": "custom.pending",
		"KAFKA_TOPIC_FRAUD_EVENTS":         "custom.fraud",
		"DATABASE_HOST":                    "db.example.com",
		"DATABASE_PORT":                    "5432",
		"DATABASE_USER":                    "admin",
		"DATABASE_PASSWORD":                "secret",
		"DATABASE_NAME":                    "fintech",
		"DATABASE_MAX_OPEN_CONNS":          "50",
		"DATABASE_MAX_IDLE_CONNS":          "10",
		"DATABASE_CONN_MAX_LIFETIME_SEC":   "3600",
		"DATABASE_CONN_MAX_IDLE_TIME_SEC":  "300",
		"REDIS_ADDR":                       "redis:6379",
		"REDIS_PASSWORD":                   "redispass",
		"REDIS_DB":                         "3",
		"REDIS_POOL_SIZE":                  "20",
	}

	for k, v := range envs {
		setEnv(t, k, v)
	}

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.App.Name != "testapp" {
		t.Errorf("App.Name = %q, want %q", cfg.App.Name, "testapp")
	}
	if cfg.App.Env != "production" {
		t.Errorf("App.Env = %q, want %q", cfg.App.Env, "production")
	}
	if cfg.App.Port != 8080 {
		t.Errorf("App.Port = %d, want 8080", cfg.App.Port)
	}
	if cfg.App.Metrics != ":9090" {
		t.Errorf("App.Metrics = %q, want %q", cfg.App.Metrics, ":9090")
	}
	if len(cfg.Kafka.Brokers) != 2 || cfg.Kafka.Brokers[0] != "broker1:9092" || cfg.Kafka.Brokers[1] != "broker2:9092" {
		t.Errorf("Kafka.Brokers = %v, want [broker1:9092 broker2:9092]", cfg.Kafka.Brokers)
	}
	if cfg.Kafka.ConsumerGroup != "my-group" {
		t.Errorf("Kafka.ConsumerGroup = %q, want %q", cfg.Kafka.ConsumerGroup, "my-group")
	}
	if cfg.Kafka.Topics.TransactionsPending != "custom.pending" {
		t.Errorf("Topics.TransactionsPending = %q, want %q", cfg.Kafka.Topics.TransactionsPending, "custom.pending")
	}
	if cfg.Kafka.Topics.FraudEvents != "custom.fraud" {
		t.Errorf("Topics.FraudEvents = %q, want %q", cfg.Kafka.Topics.FraudEvents, "custom.fraud")
	}
	if cfg.Database.Host != "db.example.com" {
		t.Errorf("Database.Host = %q, want %q", cfg.Database.Host, "db.example.com")
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("Database.Port = %d, want 5432", cfg.Database.Port)
	}
	if cfg.Database.User != "admin" {
		t.Errorf("Database.User = %q, want %q", cfg.Database.User, "admin")
	}
	if cfg.Database.Password != "secret" {
		t.Errorf("Database.Password = %q, want %q", cfg.Database.Password, "secret")
	}
	if cfg.Database.Name != "fintech" {
		t.Errorf("Database.Name = %q, want %q", cfg.Database.Name, "fintech")
	}
	if cfg.Database.MaxOpenConns != 50 {
		t.Errorf("Database.MaxOpenConns = %d, want 50", cfg.Database.MaxOpenConns)
	}
	if cfg.Database.MaxIdleConns != 10 {
		t.Errorf("Database.MaxIdleConns = %d, want 10", cfg.Database.MaxIdleConns)
	}
	if cfg.Database.ConnMaxLifetime != 3600 {
		t.Errorf("Database.ConnMaxLifetime = %d, want 3600", cfg.Database.ConnMaxLifetime)
	}
	if cfg.Database.ConnMaxIdleTime != 300 {
		t.Errorf("Database.ConnMaxIdleTime = %d, want 300", cfg.Database.ConnMaxIdleTime)
	}
	if cfg.Redis.Addr != "redis:6379" {
		t.Errorf("Redis.Addr = %q, want %q", cfg.Redis.Addr, "redis:6379")
	}
	if cfg.Redis.Password != "redispass" {
		t.Errorf("Redis.Password = %q, want %q", cfg.Redis.Password, "redispass")
	}
	if cfg.Redis.DB != 3 {
		t.Errorf("Redis.DB = %d, want 3", cfg.Redis.DB)
	}
	if cfg.Redis.PoolSize != 20 {
		t.Errorf("Redis.PoolSize = %d, want 20", cfg.Redis.PoolSize)
	}
}

func TestLoadFromFile(t *testing.T) {
	yaml := `
app:
  name: fileapp
  env: staging
  port: 3000
  metrics: ":7070"
kafka:
  brokers:
    - broker1:9092
    - broker2:9092
    - broker3:9092
  consumer_group: file-group
  topics:
    transactions_pending: file.pending
    fraud_events: file.fraud
database:
  host: file-db
  port: 5433
  user: fileuser
  password: filepass
  name: filedb
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 1800
  conn_max_idle_time: 150
redis:
  addr: file-redis:6379
  password: filepass
  db: 1
  pool_size: 15
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.App.Name != "fileapp" {
		t.Errorf("App.Name = %q, want %q", cfg.App.Name, "fileapp")
	}
	if cfg.App.Env != "staging" {
		t.Errorf("App.Env = %q, want %q", cfg.App.Env, "staging")
	}
	if cfg.App.Port != 3000 {
		t.Errorf("App.Port = %d, want 3000", cfg.App.Port)
	}
	if cfg.App.Metrics != ":7070" {
		t.Errorf("App.Metrics = %q, want %q", cfg.App.Metrics, ":7070")
	}
	if len(cfg.Kafka.Brokers) != 3 {
		t.Errorf("Kafka.Brokers length = %d, want 3", len(cfg.Kafka.Brokers))
	}
	if cfg.Kafka.ConsumerGroup != "file-group" {
		t.Errorf("Kafka.ConsumerGroup = %q, want %q", cfg.Kafka.ConsumerGroup, "file-group")
	}
	if cfg.Kafka.Topics.TransactionsPending != "file.pending" {
		t.Errorf("Topics.TransactionsPending = %q, want %q", cfg.Kafka.Topics.TransactionsPending, "file.pending")
	}
	if cfg.Kafka.Topics.FraudEvents != "file.fraud" {
		t.Errorf("Topics.FraudEvents = %q, want %q", cfg.Kafka.Topics.FraudEvents, "file.fraud")
	}
	if cfg.Database.Host != "file-db" {
		t.Errorf("Database.Host = %q, want %q", cfg.Database.Host, "file-db")
	}
	if cfg.Database.Port != 5433 {
		t.Errorf("Database.Port = %d, want 5433", cfg.Database.Port)
	}
	if cfg.Database.User != "fileuser" {
		t.Errorf("Database.User = %q, want %q", cfg.Database.User, "fileuser")
	}
	if cfg.Database.Password != "filepass" {
		t.Errorf("Database.Password = %q, want %q", cfg.Database.Password, "filepass")
	}
	if cfg.Database.Name != "filedb" {
		t.Errorf("Database.Name = %q, want %q", cfg.Database.Name, "filedb")
	}
	if cfg.Database.MaxOpenConns != 25 {
		t.Errorf("Database.MaxOpenConns = %d, want 25", cfg.Database.MaxOpenConns)
	}
	if cfg.Database.MaxIdleConns != 5 {
		t.Errorf("Database.MaxIdleConns = %d, want 5", cfg.Database.MaxIdleConns)
	}
	if cfg.Database.ConnMaxLifetime != 1800 {
		t.Errorf("Database.ConnMaxLifetime = %d, want 1800", cfg.Database.ConnMaxLifetime)
	}
	if cfg.Database.ConnMaxIdleTime != 150 {
		t.Errorf("Database.ConnMaxIdleTime = %d, want 150", cfg.Database.ConnMaxIdleTime)
	}
	if cfg.Redis.Addr != "file-redis:6379" {
		t.Errorf("Redis.Addr = %q, want %q", cfg.Redis.Addr, "file-redis:6379")
	}
	if cfg.Redis.Password != "filepass" {
		t.Errorf("Redis.Password = %q, want %q", cfg.Redis.Password, "filepass")
	}
	if cfg.Redis.DB != 1 {
		t.Errorf("Redis.DB = %d, want 1", cfg.Redis.DB)
	}
	if cfg.Redis.PoolSize != 15 {
		t.Errorf("Redis.PoolSize = %d, want 15", cfg.Redis.PoolSize)
	}
}

func TestLoadFileWithEnvOverride(t *testing.T) {
	yaml := `
app:
  name: fileapp
  port: 3000
database:
  host: file-db
  port: 5433
redis:
  addr: file-redis:6379
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	setEnv(t, "APP_NAME", "envapp")
	setEnv(t, "APP_PORT", "9999")
	setEnv(t, "DATABASE_HOST", "env-db")
	setEnv(t, "DATABASE_PORT", "3306")
	setEnv(t, "REDIS_ADDR", "env-redis:6379")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.App.Name != "envapp" {
		t.Errorf("App.Name = %q, want %q (env override)", cfg.App.Name, "envapp")
	}
	if cfg.App.Port != 9999 {
		t.Errorf("App.Port = %d, want 9999 (env override)", cfg.App.Port)
	}
	if cfg.Database.Host != "env-db" {
		t.Errorf("Database.Host = %q, want %q (env override)", cfg.Database.Host, "env-db")
	}
	if cfg.Database.Port != 3306 {
		t.Errorf("Database.Port = %d, want 3306 (env override)", cfg.Database.Port)
	}
	if cfg.Redis.Addr != "env-redis:6379" {
		t.Errorf("Redis.Addr = %q, want %q (env override)", cfg.Redis.Addr, "env-redis:6379")
	}
}

func TestTopicsConfigDefaults(t *testing.T) {
	tests := []struct {
		name   string
		input  TopicsConfig
		expect TopicsConfig
	}{
		{
			name:  "empty fills all defaults",
			input: TopicsConfig{},
			expect: TopicsConfig{
				TransactionsPending:    "transactions.pending",
				TransactionsAuthorized: "transactions.authorized",
				TransactionsSettled:    "transactions.settled",
				TransactionsFailed:     "transactions.failed",
				TransactionsReversed:   "transactions.reversed",
				FraudEvents:            "fraud.events",
				AuditLog:               "audit.log",
				NotificationsOutbox:    "notifications.outbox",
				NotificationsSent:      "notifications.sent",
				NotificationsFailed:    "notifications.failed",
			},
		},
		{
			name: "non-empty fields preserved",
			input: TopicsConfig{
				TransactionsPending: "custom.pending",
				FraudEvents:         "custom.fraud",
			},
			expect: TopicsConfig{
				TransactionsPending:    "custom.pending",
				TransactionsAuthorized: "transactions.authorized",
				TransactionsSettled:    "transactions.settled",
				TransactionsFailed:     "transactions.failed",
				TransactionsReversed:   "transactions.reversed",
				FraudEvents:            "custom.fraud",
				AuditLog:               "audit.log",
				NotificationsOutbox:    "notifications.outbox",
				NotificationsSent:      "notifications.sent",
				NotificationsFailed:    "notifications.failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.Defaults()
			if got != tt.expect {
				t.Errorf("Defaults() = %+v, want %+v", got, tt.expect)
			}
		})
	}
}

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect []string
	}{
		{
			name:   "single value",
			input:  "broker1:9092",
			expect: []string{"broker1:9092"},
		},
		{
			name:   "multiple values",
			input:  "broker1:9092,broker2:9092,broker3:9092",
			expect: []string{"broker1:9092", "broker2:9092", "broker3:9092"},
		},
		{
			name:   "values with spaces",
			input:  "broker1:9092 , broker2:9092 , broker3:9092",
			expect: []string{"broker1:9092", "broker2:9092", "broker3:9092"},
		},
		{
			name:   "leading and trailing commas",
			input:  ",broker1:9092,broker2:9092,",
			expect: []string{"broker1:9092", "broker2:9092"},
		},
		{
			name:   "empty string",
			input:  "",
			expect: []string{},
		},
		{
			name:   "only commas",
			input:  ",,,",
			expect: []string{},
		},
		{
			name:   "mixed empty and values",
			input:  "a,,b,,c",
			expect: []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitCSV(tt.input)
			if len(got) != len(tt.expect) {
				t.Fatalf("splitCSV(%q) returned %d items, want %d: got %v", tt.input, len(got), len(tt.expect), got)
			}
			for i := range got {
				if got[i] != tt.expect[i] {
					t.Errorf("splitCSV(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.expect[i])
				}
			}
		})
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	t.Run("APP_NAME", func(t *testing.T) {
		setEnv(t, "APP_NAME", "test")
		cfg := &Config{}
		applyEnvOverrides(cfg)
		if cfg.App.Name != "test" {
			t.Errorf("App.Name = %q, want %q", cfg.App.Name, "test")
		}
	})

	t.Run("APP_ENV", func(t *testing.T) {
		setEnv(t, "APP_ENV", "production")
		cfg := &Config{}
		applyEnvOverrides(cfg)
		if cfg.App.Env != "production" {
			t.Errorf("App.Env = %q, want %q", cfg.App.Env, "production")
		}
	})

	t.Run("APP_PORT", func(t *testing.T) {
		setEnv(t, "APP_PORT", "4000")
		cfg := &Config{}
		applyEnvOverrides(cfg)
		if cfg.App.Port != 4000 {
			t.Errorf("App.Port = %d, want 4000", cfg.App.Port)
		}
	})

	t.Run("APP_PORT invalid", func(t *testing.T) {
		setEnv(t, "APP_PORT", "notanumber")
		cfg := &Config{App: AppConfig{Port: 100}}
		applyEnvOverrides(cfg)
		if cfg.App.Port != 100 {
			t.Errorf("App.Port = %d, want 100 (invalid env ignored)", cfg.App.Port)
		}
	})

	t.Run("APP_METRICS", func(t *testing.T) {
		setEnv(t, "APP_METRICS", ":8080")
		cfg := &Config{}
		applyEnvOverrides(cfg)
		if cfg.App.Metrics != ":8080" {
			t.Errorf("App.Metrics = %q, want %q", cfg.App.Metrics, ":8080")
		}
	})

	t.Run("KAFKA_BROKERS", func(t *testing.T) {
		setEnv(t, "KAFKA_BROKERS", "a:9092,b:9092")
		cfg := &Config{}
		applyEnvOverrides(cfg)
		if len(cfg.Kafka.Brokers) != 2 || cfg.Kafka.Brokers[0] != "a:9092" {
			t.Errorf("Kafka.Brokers = %v, want [a:9092 b:9092]", cfg.Kafka.Brokers)
		}
	})

	t.Run("KAFKA_CONSUMER_GROUP", func(t *testing.T) {
		setEnv(t, "KAFKA_CONSUMER_GROUP", "grp")
		cfg := &Config{}
		applyEnvOverrides(cfg)
		if cfg.Kafka.ConsumerGroup != "grp" {
			t.Errorf("Kafka.ConsumerGroup = %q, want %q", cfg.Kafka.ConsumerGroup, "grp")
		}
	})

	t.Run("KAFKA_TOPIC_TRANSACTIONS_PENDING", func(t *testing.T) {
		setEnv(t, "KAFKA_TOPIC_TRANSACTIONS_PENDING", "my.topic")
		cfg := &Config{}
		applyEnvOverrides(cfg)
		if cfg.Kafka.Topics.TransactionsPending != "my.topic" {
			t.Errorf("Topics.TransactionsPending = %q, want %q", cfg.Kafka.Topics.TransactionsPending, "my.topic")
		}
	})

	t.Run("DATABASE_HOST", func(t *testing.T) {
		setEnv(t, "DATABASE_HOST", "myhost")
		cfg := &Config{}
		applyEnvOverrides(cfg)
		if cfg.Database.Host != "myhost" {
			t.Errorf("Database.Host = %q, want %q", cfg.Database.Host, "myhost")
		}
	})

	t.Run("DATABASE_PORT", func(t *testing.T) {
		setEnv(t, "DATABASE_PORT", "5433")
		cfg := &Config{}
		applyEnvOverrides(cfg)
		if cfg.Database.Port != 5433 {
			t.Errorf("Database.Port = %d, want 5433", cfg.Database.Port)
		}
	})

	t.Run("DATABASE_PORT invalid", func(t *testing.T) {
		setEnv(t, "DATABASE_PORT", "abc")
		cfg := &Config{Database: DatabaseConfig{Port: 5432}}
		applyEnvOverrides(cfg)
		if cfg.Database.Port != 5432 {
			t.Errorf("Database.Port = %d, want 5432 (invalid env ignored)", cfg.Database.Port)
		}
	})

	t.Run("DATABASE_USER", func(t *testing.T) {
		setEnv(t, "DATABASE_USER", "user")
		cfg := &Config{}
		applyEnvOverrides(cfg)
		if cfg.Database.User != "user" {
			t.Errorf("Database.User = %q, want %q", cfg.Database.User, "user")
		}
	})

	t.Run("DATABASE_PASSWORD", func(t *testing.T) {
		setEnv(t, "DATABASE_PASSWORD", "pass")
		cfg := &Config{}
		applyEnvOverrides(cfg)
		if cfg.Database.Password != "pass" {
			t.Errorf("Database.Password = %q, want %q", cfg.Database.Password, "pass")
		}
	})

	t.Run("DATABASE_NAME", func(t *testing.T) {
		setEnv(t, "DATABASE_NAME", "mydb")
		cfg := &Config{}
		applyEnvOverrides(cfg)
		if cfg.Database.Name != "mydb" {
			t.Errorf("Database.Name = %q, want %q", cfg.Database.Name, "mydb")
		}
	})

	t.Run("DATABASE_MAX_OPEN_CONNS", func(t *testing.T) {
		setEnv(t, "DATABASE_MAX_OPEN_CONNS", "30")
		cfg := &Config{}
		applyEnvOverrides(cfg)
		if cfg.Database.MaxOpenConns != 30 {
			t.Errorf("Database.MaxOpenConns = %d, want 30", cfg.Database.MaxOpenConns)
		}
	})

	t.Run("DATABASE_MAX_IDLE_CONNS", func(t *testing.T) {
		setEnv(t, "DATABASE_MAX_IDLE_CONNS", "15")
		cfg := &Config{}
		applyEnvOverrides(cfg)
		if cfg.Database.MaxIdleConns != 15 {
			t.Errorf("Database.MaxIdleConns = %d, want 15", cfg.Database.MaxIdleConns)
		}
	})

	t.Run("DATABASE_CONN_MAX_LIFETIME_SEC", func(t *testing.T) {
		setEnv(t, "DATABASE_CONN_MAX_LIFETIME_SEC", "7200")
		cfg := &Config{}
		applyEnvOverrides(cfg)
		if cfg.Database.ConnMaxLifetime != 7200 {
			t.Errorf("Database.ConnMaxLifetime = %d, want 7200", cfg.Database.ConnMaxLifetime)
		}
	})

	t.Run("DATABASE_CONN_MAX_IDLE_TIME_SEC", func(t *testing.T) {
		setEnv(t, "DATABASE_CONN_MAX_IDLE_TIME_SEC", "600")
		cfg := &Config{}
		applyEnvOverrides(cfg)
		if cfg.Database.ConnMaxIdleTime != 600 {
			t.Errorf("Database.ConnMaxIdleTime = %d, want 600", cfg.Database.ConnMaxIdleTime)
		}
	})

	t.Run("REDIS_ADDR", func(t *testing.T) {
		setEnv(t, "REDIS_ADDR", "r:6379")
		cfg := &Config{}
		applyEnvOverrides(cfg)
		if cfg.Redis.Addr != "r:6379" {
			t.Errorf("Redis.Addr = %q, want %q", cfg.Redis.Addr, "r:6379")
		}
	})

	t.Run("REDIS_PASSWORD", func(t *testing.T) {
		setEnv(t, "REDIS_PASSWORD", "rp")
		cfg := &Config{}
		applyEnvOverrides(cfg)
		if cfg.Redis.Password != "rp" {
			t.Errorf("Redis.Password = %q, want %q", cfg.Redis.Password, "rp")
		}
	})

	t.Run("REDIS_DB", func(t *testing.T) {
		setEnv(t, "REDIS_DB", "5")
		cfg := &Config{}
		applyEnvOverrides(cfg)
		if cfg.Redis.DB != 5 {
			t.Errorf("Redis.DB = %d, want 5", cfg.Redis.DB)
		}
	})

	t.Run("REDIS_DB invalid", func(t *testing.T) {
		setEnv(t, "REDIS_DB", "xyz")
		cfg := &Config{Redis: RedisConfig{DB: 1}}
		applyEnvOverrides(cfg)
		if cfg.Redis.DB != 1 {
			t.Errorf("Redis.DB = %d, want 1 (invalid env ignored)", cfg.Redis.DB)
		}
	})

	t.Run("REDIS_POOL_SIZE", func(t *testing.T) {
		setEnv(t, "REDIS_POOL_SIZE", "25")
		cfg := &Config{}
		applyEnvOverrides(cfg)
		if cfg.Redis.PoolSize != 25 {
			t.Errorf("Redis.PoolSize = %d, want 25", cfg.Redis.PoolSize)
		}
	})
}

func TestLoadNonExistentFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg should not be nil")
	}
	if cfg.App.Name != "" {
		t.Errorf("App.Name = %q, want empty", cfg.App.Name)
	}
	if cfg.Kafka.Topics.TransactionsPending != "transactions.pending" {
		t.Errorf("Topics.TransactionsPending = %q, want %q", cfg.Kafka.Topics.TransactionsPending, "transactions.pending")
	}
}
