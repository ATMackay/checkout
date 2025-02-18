package service

import (
	"os"

	"github.com/ATMackay/checkout/constants"
	"github.com/ATMackay/checkout/database"
	"github.com/ATMackay/checkout/server"
	"github.com/sirupsen/logrus"
)

// Service is the main application struct containing a Database,
// the http server and logger. It can be called to start and stop.
type Service struct {
	server *server.HTTPServer
	logger logrus.FieldLogger
}

// New constructs a Service with ethclient, logger and http server.
func New(port int, l logrus.FieldLogger, db database.Database, authPsswd string) *Service {
	srv := &Service{
		logger: l,
	}
	httpSrv := server.NewHTTPServer(port, l, db, authPsswd)
	srv.server = httpSrv
	return srv
}

// Start spawns the HTTP server.
func (s *Service) Start() {
	s.logger.Infof("starting %s", constants.ServiceName)
	s.logger.WithFields(logrus.Fields{
		"compilation_date": constants.BuildDate,
		"commit":           constants.GitCommit,
		"version":          constants.Version,
	}).Info("version")

	s.server.Start()

	s.logger.Infof("listening on port %v", s.server.Addr())
}

// Start gracefully shutts down the HTTP server.
func (s *Service) Stop(sig os.Signal) {
	s.logger.WithField("signal", sig).Infof("stopping %v service", constants.ServiceName)

	if err := s.server.Stop(); err != nil {
		s.logger.WithField("error", err).Error("error stopping server")
	}
}

// Server exposes the http server externally.
func (s *Service) Server() *server.HTTPServer {
	return s.server
}
