package main

import (
	"github.com/ATMackay/checkout/cmd"

	log "github.com/sirupsen/logrus"
)

func main() {
	command := cmd.NewCheckoutCmd()
	if err := command.Execute(); err != nil {
		log.WithError(err).Fatalf("main: execution failed")
	}
}
