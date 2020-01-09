package rpc

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"testing"
	"time"
)

func TestNewRpcServer(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)
	logFile, err := os.OpenFile("hihi.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	fmt.Println("Will be logged to stdout and ", logFile)
	writer := io.MultiWriter(logFile)
	logrus.SetOutput(writer)
	rpc := NewRpcServer("9090")
	rpc.Start()
	time.Sleep(time.Second * 20)
}
