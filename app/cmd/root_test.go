package cmd

import (
	"testing"
	log "github.com/sirupsen/logrus"
)

func TestLogger(t *testing.T) {
	initLogger()
	log.Debug("Test Debug")
	log.Info("Test Info")
	//log.Warn("Test Warn")
	//log.Fatal("Test Fatal")
}
