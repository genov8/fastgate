package config

import (
	"errors"
	"gopkg.in/yaml.v2"
	"log"
	"os"
)

type Config struct {
	Aggregations []Aggregation `yaml:"aggregations"`
}

type Aggregation struct {
	Path      string    `yaml:"path"`
	Calls     []Call    `yaml:"calls"`
	Response  Mapping   `yaml:"response"`
	RateLimit RateLimit `yaml:"rate_limit,omitempty"`
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

type RateLimit struct {
	Limit    int `yaml:"limit"`
	Interval int `yaml:"interval"`
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

	if err := validateConfig(&cfg); err != nil {
		log.Fatal("Configuration validation error:", err)
		return nil, err
	}

	log.Println("Configuration loaded successfully")
	return &cfg, nil
}

func validateConfig(cfg *Config) error {
	if len(cfg.Aggregations) == 0 {
		return errors.New("at least one aggregation must be defined")
	}

	for _, agg := range cfg.Aggregations {
		if agg.Path == "" {
			return errors.New("path is required for each aggregation")
		}
		if len(agg.Calls) == 0 {
			return errors.New("at least one call must be defined for each aggregation")
		}

		callNames := make(map[string]bool)
		for _, call := range agg.Calls {
			if call.Name == "" {
				return errors.New("name is required for each call")
			}
			if call.Backend == "" {
				return errors.New("backend is required for each call")
			}
			callNames[call.Name] = true

			for _, value := range call.Params {
				if !(isValidParam(value)) {
					return errors.New("invalid parameter reference: " + value + " in call " + call.Name)
				}
			}
		}

		if (agg.RateLimit.Limit > 0 && agg.RateLimit.Interval == 0) ||
			(agg.RateLimit.Limit == 0 && agg.RateLimit.Interval > 0) {
			return errors.New("both 'limit' and 'interval' must be set in rate_limit")
		}

		if len(agg.Response.Structure) > 0 {
			for key := range agg.Response.Structure {
				if !callNames[key] && key != "error" {
					return errors.New("invalid response mapping: '" + key + "' does not match any service call or 'error'")
				}
			}
		}
	}

	return nil
}

func isValidParam(param string) bool {
	return param == "" ||
		(len(param) > 6 && (param[:6] == "$path." || param[:7] == "$query." || param[:8] == "$header."))
}
