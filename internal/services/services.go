package services

import (
	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/internal/services/shttp"
)

var logger = logging.New().WithName("services")

// InitServices initialise the service system
func InitServices(cfg config.Config) error {
	// here you can add more services
	/*
		_, err := sconfig.NewSConfig()
		if err != nil {
			return err
		}
	*/
	_, err := shttp.NewSHttp(cfg)
	if err != nil {
		return err
	}

	return nil
}
