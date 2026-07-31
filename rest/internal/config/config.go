package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Env            string        `yaml:"env"`
	MigrationsPath string        `yaml:"migrations_path" env-required:"true"`
	PublicKeyPath  string        `yaml:"public_key_path" env-required:"true"`
	Clients        ClientsConfig `yaml:"clients"`
	HTTPServer     `yaml:"http_server"`
	Database       `yaml:"database"`
}

type HTTPServer struct {
	Address         string        `yaml:"address" env-default:"localhost:8080"`
	Timeout         time.Duration `yaml:"timeout" env-default:"5s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" env-default:"60s"`
	ShutDownTimeout time.Duration `yaml:"shutdown_timeount" env-default:"10s"`
}

type Database struct {
	Host     string `yaml:"host" env-required:"true" env:"DB_HOST"`
	Port     int    `yaml:"port" env-required:"true" env:"DB_PORT"`
	User     string `yaml:"user" env-required:"true" env:"DB_USER"`
	Password string `yaml:"password" env-required:"true" env:"DB_PASSWORD"`
	Name     string `yaml:"name" env-required:"true" env:"DB_NAME"`
	SSLMode  string `yaml:"sslmode" env-default:"disable" env:"DB_SSLMODE"`
}

type ClientSSO struct {
	Address      string        `yaml:"address"`
	Timeout      time.Duration `yaml:"timeout"`
	RetriesCount int           `yaml:"retries_count"`
}

type ClientKafka struct {
	Address       string        `yaml:"address" env-required:"true"`
	Topic         string        `yaml:"topic" env-required:"true"`
	GroupID       string        `yaml:"group_id" env-default:"rest"`
	TopicTimeout  time.Duration `yaml:"topic_timeout" env-required:"true"`
	NumPartitions int           `yaml:"num_partitions" env-required:"true"`
}

type ClientsConfig struct {
	SSO   ClientSSO   `yaml:"sso"`
	Kafka ClientKafka `yaml:"kafka"`
}

func MustLoad() *Config {
	if err := godotenv.Load(); err != nil {
		panic("Error loading .env file")
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		panic(fmt.Sprintf("Config path %s not found", configPath))
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("failed to read config: " + err.Error())
	}

	return &cfg
}
