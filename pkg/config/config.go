package config

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	DB DBConfig
}

type DBConfig struct {
	User     string `envconfig:"DB_USER"     required:"true"`
	Password string `envconfig:"DB_PASSWORD" required:"true"`
	Host     string `envconfig:"DB_HOST"     default:"localhost"`
	Port     string `envconfig:"DB_PORT"     default:"5432"`
	Name     string `envconfig:"DB_NAME"     required:"true"`
	SSLMode  string `envconfig:"DB_SSLMODE"  default:"disable"`
}

func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Name,
		c.SSLMode,
	)
}

func (c DBConfig) validate() error {
	var errs []error

	if c.User == "" {
		errs = append(errs, errors.New("DB_USER is required"))
	}
	if c.Password == "" {
		errs = append(errs, errors.New("DB_PASSWORD is required"))
	}
	if c.Host == "" {
		errs = append(errs, errors.New("DB_HOST is required"))
	}
	if c.Name == "" {
		errs = append(errs, errors.New("DB_NAME is required"))
	}

	port, err := strconv.Atoi(c.Port)
	if err != nil || port < 1 || port > 65535 {
		errs = append(errs, fmt.Errorf("DB_PORT must be a valid port number (1-65535), got %q", c.Port))
	}

	validSSLModes := map[string]bool{
		"disable": true, "require": true, "verify-ca": true, "verify-full": true,
	}
	if !validSSLModes[c.SSLMode] {
		errs = append(errs, fmt.Errorf("DB_SSLMODE must be one of disable|require|verify-ca|verify-full, got %q", c.SSLMode))
	}

	return errors.Join(errs...)
}

func Load(envFiles ...string) (*Config, error) {
	// Best-effort: load .env files if present; real env vars always take precedence.
	_ = godotenv.Load(envFiles...)

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	if err := cfg.DB.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}
