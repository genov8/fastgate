package config

import (
	"gopkg.in/yaml.v2"
	"log"
	"os"
)

type Config struct {
	Aggregations []Aggregation `yaml:"aggregations"`
}

type Aggregation struct {
	Path     string  `yaml:"path"`
	Calls    []Call  `yaml:"calls"`
	Response Mapping `yaml:"response"`
}

type Call struct {
	Name     string            `yaml:"name"`
	Backend  string            `yaml:"backend"`
	Required bool              `yaml:"required,omitempty"`
	Params   map[string]string `yaml:"params,omitempty"`
}

type Mapping struct {
	Structure map[string]string `yaml:"structure"`
}

func LoadConfig(filename string) (*Config, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal("Error loading configuration:", err)
		return nil, err
	}
	var cfg Config
	err = yaml.Unmarshal(file, &cfg)
	if err != nil {
		log.Fatal("YAML parsing error:", err)
		return nil, err
	}
	log.Println("Configuration loaded successfully")
	return &cfg, nil
}
