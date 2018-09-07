package cmd

import (
	log "github.com/sirupsen/logrus"
	"testing"
)

func TestLogger(t *testing.T) {
	initLogger()
	log.Debug("Test Debug")
	log.Info("Test Info")
	//log.Warn("Test Warn")
	//log.Fatal("Test Fatal")
}
