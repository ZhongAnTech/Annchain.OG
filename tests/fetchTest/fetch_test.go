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
package fetchTest

import (
	"fmt"
	"testing"
)

func CalculateRequestSpan(remoteHeight, localHeight uint64) (int64, int, int, uint64) {
	var (
		from     int
		count    int
		MaxCount = 192 / 16
	)
	// requestHead is the highest block that we will ask for. If requestHead is not offset,
	// the highest block that we will get is 16 blocks back from head, which means we
	// will fetch 14 or 15 blocks unnecessarily in the case the height difference
	// between us and the peer is 1-2 blocks, which is most common
	requestHead := int(remoteHeight) - 1
	if requestHead < 0 {
		requestHead = 0
	}
	// requestBottom is the lowest block we want included in the query
	// Ideally, we want to include just below own head
	requestBottom := int(localHeight - 1)
	if requestBottom < 0 {
		requestBottom = 0
	}
	totalSpan := requestHead - requestBottom
	span := 1 + totalSpan/MaxCount
	fmt.Println(span)
	if span < 2 {
		span = 2
	}
	if span > 16 {
		span = 16
	}
	fmt.Println(span)
	count = 1 + totalSpan/span
	if count > MaxCount {
		count = MaxCount
	}
	fmt.Println(count, MaxCount)
	if count < 2 {
		count = 2
	}
	fmt.Println(count, MaxCount)
	from = requestHead - (count-1)*span
	if from < 0 {
		from = 0
	}
	max := from + (count-1)*span
	return int64(from), count, span - 1, uint64(max)
}

func TestCalculateRequestSpan(t *testing.T) {
	from, count, skip, max := CalculateRequestSpan(8, 5)
	fmt.Println(from, count, skip, max)
	from, count, skip, max = CalculateRequestSpan(10, 0)
	fmt.Println(from, count, skip, max)
	from, count, skip, max = CalculateRequestSpan(389, 241)
	fmt.Println(from, count, skip, max)
	from, count = calculate(8, 5)
	fmt.Println(from, count, 15)
	from, count = calculate(10, 0)
	fmt.Println(from, count, 15)
	from, count = calculate(389, 241)
	fmt.Println(from, count, 15)
}

func calculate(remote, local uint64) (int64, int) {
	// Request the topmost blocks to short circuit binary ancestor lookup
	head := local
	if head > remote {
		head = remote
	}
	from := int64(head) - 192
	if from < 0 {
		from = 0
	}
	// Span out with 15 block gaps into the future to catch bad head reports
	limit := 2 * 192 / 16
	count := 1 + int((int64(local)-from)/16)
	if count > limit {
		count = limit
	}
	return from, count
}


func TestUint (t *testing.T){
	floor, ceil := int64(-1), uint64(50)
	height:= uint64(50)
	// Request the topmost blocks to short circuit binary ancestor lookup
	head := ceil
	if head > height {
		head = height
	}
	from := int64(head) - int64(192)
	fmt.Println(uint64(from),from)
	if from < 0 {
		from = 0
	}
	// Span out with 15 block gaps into the future to catch bad head reports
	limit := 2 * 192 / 16
	count := 1 + int((int64(ceil)-from)/16)
	if count > limit {
		count = limit
	}
	fmt.Println(uint64(from), count, 15, false,floor)
}