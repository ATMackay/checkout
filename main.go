package main

import (
	"log/slog"
	"os"

	"github.com/ATMackay/checkout/cmd"
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
		slog.Error("main: execution failed", "error", err)
		os.Exit(1)
	}
}
