package fastgate

import (
	"fastgate/internal/config"
	"fastgate/internal/gateway"
)

func New(configPath string) (*gateway.Router, error) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	return gateway.NewRouter(cfg), nil
}
