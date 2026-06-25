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
// (d DatabaseConfig) is the RECEIVER - it means "this function belongs to DatabaseConfig"
// It builds a PostgreSQL connection string like
// postgres://user:password@host:port/dbname?sslmode=disable
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
// It returns a pointer to Config (*Config) and an error.
//
// WHY A POINTER?
// Returning *Config (pointer) instead of Config (value) means:
// - We avoid copying the entire struct (performance)
// - The caller gets a reference to the same data
// - Convention in Go for "constructed" objects
func Load() (*Config, error) {
	// Create a new Koanf instance with "." as the key delimiter.
	// This means "SERVER_PORT" becomes "server.port" after transformation.
	k := koanf.New(".")

	// Step 1: Load from .env file
	// The dotenv parser reads KEY=VALUE pairs from the file.
	// We use the same callback to transform keys (see Step 2 explanation).
	if err := k.Load(file.Provider(".env"), dotenv.ParserEnv("", ".", func(s string) string {
		// Transform "SERVER_PORT" → "server.port"
		// 1. Lowercase:    "server_port"
		// 2. Replace _ with .: "server.port"
		return strings.Replace(strings.ToLower(s), "_", ".", -1)
	})); err != nil {
		// If .env file doesn't exist, that's okay in production.
		// We just log it and continue (env vars will provide values).
		fmt.Printf("⚠️  No .env file found, using environment variables only: %v\n", err)
	}

	// Step 2: Load from actual environment variables (overrides .env)
	// The first arg "" means no prefix filter (read ALL env vars).
	// The second arg "." is the delimiter for nested keys.
	// The callback transforms env var names to koanf key format.
	if err := k.Load(env.Provider("", ".", func(s string) string {
		return strings.Replace(strings.ToLower(s), "_", ".", -1)
	}), nil); err != nil {
		return nil, fmt.Errorf("error loading env variables: %w", err)
	}

	// Step 3: Unmarshal into our typed Config struct
	// This converts the flat map of key-value pairs into nested structs.
	// "Unmarshal" = converting raw data into a structured type.
	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	return &cfg, nil
}
