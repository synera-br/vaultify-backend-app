package config

import (
	"errors"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
type Config struct {
	Port                             string `mapstructure:"PORT"`
	GinMode                          string `mapstructure:"GIN_MODE"`
	FirebaseProjectID                string `mapstructure:"FIREBASE_PROJECT_ID"`
	GoogleApplicationCredentials     string `mapstructure:"GOOGLE_APPLICATION_CREDENTIALS"`
	FirebaseServiceAccountJSONBase64 string `mapstructure:"FIREBASE_SERVICE_ACCOUNT_JSON_BASE64"`
	EncryptionKey                    string `mapstructure:"ENCRYPTION_KEY"` // Base64 encoded
	StripeSecretKey                  string `mapstructure:"STRIPE_SECRET_KEY"`
	StripeWebhookSecret              string `mapstructure:"STRIPE_WEBHOOK_SECRET"`
	ClientURL                        string `mapstructure:"CLIENT_URL"`
}

var appConfig *Config

// LoadConfig loads configuration from environment variables using Viper.
func LoadConfig() (*Config, error) {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set default values
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("GIN_MODE", "debug")

	// Bind environment variables
	viper.BindEnv("PORT")
	viper.BindEnv("GIN_MODE")
	viper.BindEnv("FIREBASE_PROJECT_ID")
	viper.BindEnv("GOOGLE_APPLICATION_CREDENTIALS")
	viper.BindEnv("FIREBASE_SERVICE_ACCOUNT_JSON_BASE64")
	viper.BindEnv("ENCRYPTION_KEY")
	viper.BindEnv("STRIPE_SECRET_KEY")
	viper.BindEnv("STRIPE_WEBHOOK_SECRET")
	viper.BindEnv("CLIENT_URL")

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, errors.New("failed to unmarshal config: " + err.Error())
	}

	// Validate required fields
	if cfg.FirebaseProjectID == "" {
		return nil, errors.New("FIREBASE_PROJECT_ID is required")
	}
	if cfg.GoogleApplicationCredentials == "" && cfg.FirebaseServiceAccountJSONBase64 == "" {
		return nil, errors.New("either GOOGLE_APPLICATION_CREDENTIALS or FIREBASE_SERVICE_ACCOUNT_JSON_BASE64 is required")
	}
	if cfg.EncryptionKey == "" {
		return nil, errors.New("ENCRYPTION_KEY is required")
	}
	if cfg.StripeSecretKey == "" {
		return nil, errors.New("STRIPE_SECRET_KEY is required")
	}
	if cfg.StripeWebhookSecret == "" {
		return nil, errors.New("STRIPE_WEBHOOK_SECRET is required")
	}
	if cfg.ClientURL == "" {
		return nil, errors.New("CLIENT_URL is required")
	}


	appConfig = &cfg
	return appConfig, nil
}

// GetConfig returns the loaded application configuration.
// It will panic if LoadConfig has not been called successfully.
func GetConfig() *Config {
	if appConfig == nil {
		// This typically means LoadConfig was not called, or it failed.
		// Depending on application design, you might want to handle this by
		// attempting to load config here, or ensuring LoadConfig is called at startup.
		panic("config not loaded; call LoadConfig first")
	}
	return appConfig
}
