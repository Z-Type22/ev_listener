package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Env           string        `yaml:"env"`
	RPCUrl        string        `yaml:"rpc_url" env-required:"true"`
	Kafka         Kafka         `yaml:"kafka"`
	ClientTimeout time.Duration `yaml:"client_timeout" env-default:"5s"`
	PathContracts PathContracts `yaml:"path_contracts"`
	ListContracts ListContracts
}

type ListContracts struct {
	SubscriptionContract string `env-required:"true" env:"SUBSCRIPTION_CONTRACT"`
	MarketplaceContract  string `env-required:"true" env:"MARKETPLACE_CONTRACT"`
	GiftContract         string `env-required:"true" env:"GIFT_CONTRACT"`
	PlanContract         string `env-required:"true" env:"PLAN_CONTRACT"`
	DonateContract       string `env-required:"true" env:"DONATE_CONTRACT"`
}

type PathContracts struct {
	SubscriptionPath string `yaml:"subscription_abi_path" env-required:"true"`
	MarketplacePath  string `yaml:"marketplace_abi_path" env-required:"true"`
	GiftPath         string `yaml:"gift_abi_path" env-required:"true"`
	PlanPath         string `yaml:"plan_abi_path" env-required:"true"`
	DonatePath       string `yaml:"donate_abi_path" env-required:"true"`
}

type Kafka struct {
	Address      string        `yaml:"address" env-required:"true"`
	Topic        string        `yaml:"topic" env-required:"true"`
	WriteTimeout time.Duration `yaml:"write_timeout" env-default:"10s"`
	MaxAttempts  int           `yaml:"max_attempts" env-required:"true"`
	BatchSize    int           `yaml:"batch_size" env-required:"true"`
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
