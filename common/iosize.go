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
package common

import (
	"fmt"
)

// StorageSize is a wrapper around a float value that supports user friendly
// formatting.
type IOSize float64

// String implements the stringer interface.
func (s IOSize) String() string {
	if s > 1048576 {
		return fmt.Sprintf("%.2f mB", s/1000000)
	} else if s > 1024 {
		return fmt.Sprintf("%.2f kB", s/1024)
	} else {
		return fmt.Sprintf("%.2f B", s)
	}
}

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (s IOSize) TerminalString() string {
	if s > 1048576 {
		return fmt.Sprintf("%.2fmB", s/1000000)
	} else if s > 1024 {
		return fmt.Sprintf("%.2fkB", s/1000)
	} else {
		return fmt.Sprintf("%.2fB", s)
	}
}
