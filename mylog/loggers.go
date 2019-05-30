// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package mylog

import (
	"fmt"
	"github.com/lestrrat/go-file-rotatelogs"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"
)

var TxLogger *logrus.Logger

func panicIfError(err error, message string) {
	if err != nil {
		fmt.Println(message)
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func RotateLog(abspath string) *rotatelogs.RotateLogs {
	logFile, err := rotatelogs.New(
		abspath+"%Y%m%d%H%M%S.log",
		rotatelogs.WithLinkName(abspath+".log"),
		rotatelogs.WithMaxAge(24*time.Hour*7),
		rotatelogs.WithRotationTime(time.Hour*24),
	)
	panicIfError(err, "err init log")
	return logFile
}

func InitLogger(logger *logrus.Logger, logdir string, outputFile string) *logrus.Logger {
	var writer io.Writer
	if logdir != "" {
		folderPath, err := filepath.Abs(logdir)
		panicIfError(err, fmt.Sprintf("Error on parsing log path: %s", logdir))

		abspath, err := filepath.Abs(path.Join(logdir, outputFile))
		panicIfError(err, fmt.Sprintf("Error on parsing log file path: %s", logdir))

		err = os.MkdirAll(folderPath, os.ModePerm)
		panicIfError(err, fmt.Sprintf("Error on creating log dir: %s", folderPath))

		logrus.WithField("path", abspath).Info("Additional logger")
		//logFile, err := os.OpenFile(abspath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		panicIfError(err, fmt.Sprintf("Error on creating log file: %s", abspath))
		//write  a message to just one  files

		writer = io.MultiWriter(logger.Out, RotateLog(abspath))
	} else {
		writer = logger.Out
	}
	return &logrus.Logger{
		Level:     logger.Level,
		Formatter: logger.Formatter,
		Out:       writer,
		Hooks:     logger.Hooks,
	}
}

func InitLoggers(logger *logrus.Logger, logdir string) {
	TxLogger = InitLogger(logger, logdir, "og_tx.log")
	logrus.Debug("Additional logger initialized.")
}
