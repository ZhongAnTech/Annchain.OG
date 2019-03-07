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
package ffchan

import (
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

func TestChan(t *testing.T) {
	c := make(chan bool)

	go func() {
		time.Sleep(time.Second * 5)
		<-c
		time.Sleep(time.Second * 5)
		<-c
	}()

	logrus.Info("Sending 1st")
	<-NewTimeoutSender(c, true).C
	logrus.Info("Sending 2st")
	<-NewTimeoutSender(c, true).C

}
