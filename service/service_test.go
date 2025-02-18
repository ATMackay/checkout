package service

import (
	"io"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

var testLog = &logrus.Logger{
	Out:       io.Discard,
	Formatter: &logrus.TextFormatter{DisableTimestamp: true},
	Level:     logrus.InfoLevel,
}

func Test_Service_Start_Stop(t *testing.T) {
	s := New(8000, testLog, nil, "")
	s.Start()
	require.NotNil(t, s.Server())
	s.Stop(os.Kill)
}
