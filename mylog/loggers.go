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
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"path"
	"path/filepath"
)

func panicIfError(err error, message string) {
	if err != nil {
		fmt.Println(message)
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func RotateLog(abspath string) *lumberjack.Logger {
	logFile := &lumberjack.Logger{
		Filename:   abspath,
		MaxSize:    10, // megabytes
		MaxBackups: 100,
		MaxAge:     90,   //days
		Compress:   true, // disabled by default
	}
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
	newLogger := &logrus.Logger{
		Level:        logger.Level,
		Formatter:    logger.Formatter,
		Out:          writer,
		Hooks:        logger.Hooks,
		ExitFunc:     logger.ExitFunc,
		ReportCaller: logger.ReportCaller,
	}
	return newLogger
}
