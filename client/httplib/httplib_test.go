package httplib

import (
	"fmt"
	"testing"
)

func TestGet(t *testing.T) {
	url := "http://127.0.0.1:8000/"
	req := NewBeegoRequest(url, "GET")
	fmt.Println(req.String())
}
