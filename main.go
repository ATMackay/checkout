package main

import (
	"github.com/ATMackay/checkout/cmd"

	log "github.com/sirupsen/logrus"
)

// @title         Checkout API
// @version       0.1.0
// @description   API for inventory, pricing and purchases
// @BasePath      /
// @schemes       http
// @host          localhost:8080

// @securityDefinitions.apikey  XAuthPassword
// @in                          header
// @name                        X-Auth-Password

func main() {
	command := cmd.NewCheckoutCmd()
	if err := command.Execute(); err != nil {
		log.WithError(err).Fatalf("main: execution failed")
	}
}
