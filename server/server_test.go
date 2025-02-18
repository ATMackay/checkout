package server

import (
	"io"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

var testLog = &logrus.Logger{
	Out:       io.Discard,
	Formatter: &logrus.TextFormatter{DisableTimestamp: true},
	Level:     logrus.InfoLevel,
}

func Test_ServerStartStop(t *testing.T) {

	s := NewHTTPServer(8000, testLog, nil, "")

	s.RegisterHandlers()

	s.Start()
	// Wait until server goroutine has initialized
	for !s.started.Load() {
	}

	require.Equal(t, ":8000", s.Addr())

	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
}
