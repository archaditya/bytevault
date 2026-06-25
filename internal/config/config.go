package config

import (
	"fmt"
	"strings"

	"github.com/knadh/koanf/parsers/dotenv"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	Server   ServerConfig   `koanf:"server"`
	Database DatabaseConfig `koanf:"db"`
	App      AppConfig      `koanf:"app"`
	JWT      JWTConfig      `koanf:"jwt"`
	Storage  StorageConfig  `koanf:"storage"`
}

type StorageConfig struct {
	Provider          string `koanf:"provider"` // local, cloudinary, r2
	LocalDir          string `koanf:"localdir"`
	CloudinaryURL     string `koanf:"cloudinaryurl"`
	R2Endpoint        string `koanf:"r2endpoint"`
	R2AccessKeyID     string `koanf:"r2accesskeyid"`
	R2SecretAccessKey string `koanf:"r2secretaccesskey"`
	R2Bucket          string `koanf:"r2bucket"`
}

type ServerConfig struct {
	Port string `koanf:"port"`
}

type DatabaseConfig struct {
	Host     string `koanf:"host"`
	Port     string `koanf:"port"`
	User     string `koanf:"user"`
	Password string `koanf:"password"`
	Name     string `koanf:"name"`
	SSLMode  string `koanf:"sslmode"`
}

// DSN (Data Source Name) is a METHOD on DatabaseConfig.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

type AppConfig struct {
	Env string `koanf:"env"`
}

type JWTConfig struct {
	Secret        string `koanf:"secret"`
	AccessExpiry  string `koanf:"accessexpiry"`
	RefreshExpiry string `koanf:"refreshexpiry"`
}

// Load reads configuration from .env file and environment variables.
func Load() (*Config, error) {
	k := koanf.New(".")

	// Custom key transformer: splits only on the first underscore
	// e.g., STORAGE_R2_ENDPOINT -> storage.r2endpoint
	transformKey := func(s string) string {
		s = strings.ToLower(s)
		parts := strings.SplitN(s, "_", 2)
		if len(parts) == 2 {
			return parts[0] + "." + strings.Replace(parts[1], "_", "", -1)
		}
		return s
	}

	// Step 1: Load from .env file
	if err := k.Load(file.Provider(".env"), dotenv.ParserEnv("", ".", transformKey)); err != nil {
		fmt.Printf("⚠️  No .env file found, using environment variables only: %v\n", err)
	}

	// Step 2: Load from actual environment variables (overrides .env)
	if err := k.Load(env.Provider("", ".", transformKey), nil); err != nil {
		return nil, fmt.Errorf("error loading env variables: %w", err)
	}

	// Step 3: Unmarshal into our typed Config struct
	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	return &cfg, nil
}
