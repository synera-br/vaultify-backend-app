package configs

import (
	"os"
	"log"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port string `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`
	Redis struct {
		Address  string `yaml:"address"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`
	Firestore struct {
		ProjectID      string `yaml:"project_id"`
		CredentialsFile string `yaml:"credentials_file"`
	} `yaml:"firestore"`
	RabbitMQ struct {
		URL      string `yaml:"url"`
		QueueName string `yaml:"queue_name"`
	} `yaml:"rabbitmq"`
}

func LoadConfig() (*Config, error) {
	var cfg Config
	configPath := os.Getenv("PATH_CONFIG")
	if configPath == "" {
		configPath = "configs/config.yaml" // Default path
	}

	log.Printf("Loading configuration from: %s", configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("Error reading config file: %v", err)
		return nil, err
	}

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Printf("Error unmarshalling config data: %v", err)
		return nil, err
	}

	return &cfg, nil
}
