package types

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestRandomArchive(t *testing.T) {
	a := RandomArchive()
	d, _ := json.MarshalIndent(a, "", "\t")
	fmt.Println(string(d))
}
