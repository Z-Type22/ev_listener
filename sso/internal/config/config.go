package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Env            string        `yaml:"env" env-default:"local"`
	PublicKeyPath  string        `yaml:"public_key_path" env-required:"true"`
	PrivateKeyPath string        `yaml:"private_key_path" env-required:"true"`
	AccessTTL      time.Duration `yaml:"access_ttl" env-required:"true"`
	RefreshTTL     time.Duration `yaml:"refresh_ttl" env-required:"true"`
	GRPC           GRPCConfig    `yaml:"grpc"`
	MigrationsPath string        `yaml:"migrations_path" env-required:"true"`
	Database
}

type GRPCConfig struct {
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

type Database struct {
	Host     string `env-required:"true" env:"DB_HOST"`
	Port     int    `env-required:"true" env:"DB_PORT"`
	User     string `env-required:"true" env:"DB_USER"`
	Password string `env-required:"true" env:"DB_PASSWORD"`
	Name     string `env-required:"true" env:"DB_NAME"`
	SSLMode  string `env-default:"disable" env:"DB_SSLMODE"`
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

func MustLoadByPath(configPath string) *Config {
	if err := godotenv.Load("../.env"); err != nil {
		panic("Error loading .env file: " + err.Error())
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
