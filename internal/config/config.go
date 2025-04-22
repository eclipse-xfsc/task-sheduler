package config

import "time"

type Config struct {
	HTTP         httpConfig
	Auth         authConfig
	Mongo        mongoConfig
	Policy       policyConfig
	Executor     executorConfig
	ListExecutor listExecutorConfig
	Cache        cacheConfig
	Metrics      metricsConfig
	OAuth        oauthConfig
	Nats         natsConfig

	LogLevel string `envconfig:"LOG_LEVEL" default:"INFO"`
}

// HTTP Server configuration
type httpConfig struct {
	Host         string        `envconfig:"HTTP_HOST"`
	Port         string        `envconfig:"HTTP_PORT" default:"8080"`
	IdleTimeout  time.Duration `envconfig:"HTTP_IDLE_TIMEOUT" default:"120s"`
	ReadTimeout  time.Duration `envconfig:"HTTP_READ_TIMEOUT" default:"10s"`
	WriteTimeout time.Duration `envconfig:"HTTP_WRITE_TIMEOUT" default:"10s"`
}

type authConfig struct {
	// Enabled specifies whether authentication is enabled
	Enabled         bool          `envconfig:"AUTH_ENABLED" default:"false"`
	JwkURL          string        `envconfig:"AUTH_JWK_URL"`
	RefreshInterval time.Duration `envconfig:"AUTH_REFRESH_INTERVAL" default:"1h"`
}

// MongoDB configuration
type mongoConfig struct {
	// Addr specifies the address of the MongoDB instance
	Addr          string `envconfig:"MONGO_ADDR" required:"true"`
	User          string `envconfig:"MONGO_USER" required:"true"`
	Pass          string `envconfig:"MONGO_PASS" required:"true"`
	AuthMechanism string `envconfig:"MONGO_AUTH_MECHANISM" default:"SCRAM-SHA-1"`
}

type policyConfig struct {
	//Addr specifies the address of the policy service
	Addr string `envconfig:"POLICY_ADDR" required:"true"`
}

type executorConfig struct {
	Workers        int           `envconfig:"EXECUTOR_WORKERS" default:"5"`
	PollInterval   time.Duration `envconfig:"EXECUTOR_POLL_INTERVAL" default:"1s"`
	MaxTaskRetries int           `envconfig:"EXECUTOR_MAX_TASK_RETRIES" default:"10"`
}

type listExecutorConfig struct {
	Workers      int           `envconfig:"LIST_EXECUTOR_WORKERS" default:"5"`
	PollInterval time.Duration `envconfig:"LIST_EXECUTOR_POLL_INTERVAL" default:"1s"`
}

type cacheConfig struct {
	//Addr specifies the address of the cache service
	Addr string `envconfig:"CACHE_ADDR" required:"true"`
}

type metricsConfig struct {
	// Addr specifies the address of the metrics endpoint
	Addr string `envconfig:"METRICS_ADDR" default:":2112"`
}

// OAuth Client configuration
type oauthConfig struct {
	ClientID     string `envconfig:"OAUTH_CLIENT_ID"`
	ClientSecret string `envconfig:"OAUTH_CLIENT_SECRET"`
	TokenURL     string `envconfig:"OAUTH_TOKEN_URL"`
}

type natsConfig struct {
	// Addr specifies the address of the NATS server
	Addr string `envconfig:"NATS_ADDR"`
	// Subject specifies the subject of the NATS subscription
	Subject string `envconfig:"NATS_SUBJECT" default:"external"`
}
