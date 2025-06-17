// Package config provides configuration management for the application.
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Webhook    WebhookConfig    `mapstructure:"webhook"`
	Scheduler  SchedulerConfig  `mapstructure:"scheduler"`
	Middleware MiddlewareConfig `mapstructure:"middleware"`
}

type ServerConfig struct {
	Port         string `mapstructure:"port"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type WebhookConfig struct {
	URL            string               `mapstructure:"url"`
	AuthKey        string               `mapstructure:"auth_key"`
	Timeout        int                  `mapstructure:"timeout"`
	CircuitBreaker CircuitBreakerConfig `mapstructure:"circuit_breaker"`
}

type CircuitBreakerConfig struct {
	MaxRequests      uint32  `mapstructure:"max_requests"`
	Interval         int     `mapstructure:"interval"`
	Timeout          int     `mapstructure:"timeout"`
	FailureRatio     float64 `mapstructure:"failure_ratio"`
	ConsecutiveFails uint32  `mapstructure:"consecutive_fails"`
}

type SchedulerConfig struct {
	IntervalMinutes int `mapstructure:"interval_minutes"`
	BatchSize       int `mapstructure:"batch_size"`
}

type MiddlewareConfig struct {
	RateLimit      int      `mapstructure:"rate_limit"`
	RateLimitBurst int      `mapstructure:"rate_limit_burst"`
	EnableCORS     bool     `mapstructure:"enable_cors"`
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.read_timeout", 10)
	viper.SetDefault("server.write_timeout", 10)
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("webhook.timeout", 30)
	viper.SetDefault("webhook.circuit_breaker.max_requests", 3)
	viper.SetDefault("webhook.circuit_breaker.interval", 60)
	viper.SetDefault("webhook.circuit_breaker.timeout", 60)
	viper.SetDefault("webhook.circuit_breaker.failure_ratio", 0.6)
	viper.SetDefault("webhook.circuit_breaker.consecutive_fails", 5)
	viper.SetDefault("scheduler.interval_minutes", 2)
	viper.SetDefault("scheduler.batch_size", 2)
	viper.SetDefault("middleware.rate_limit", 100)
	viper.SetDefault("middleware.rate_limit_burst", 1000)
	viper.SetDefault("middleware.enable_cors", true)
	viper.SetDefault("middleware.allowed_origins", []string{"*"})

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	viper.AutomaticEnv()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// GetDSN returns PostgreSQL connection string.
func (d *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}
