package dns

import (
	"fmt"
	"testing"
)

func TestDomain(t *testing.T) {
	fmt.Println(Lookup("www.baidu.com"))
}
