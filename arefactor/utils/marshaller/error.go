package marshaller

import "fmt"

func ErrorNotEnoughBytes(expect, reality int) error {
	return fmt.Errorf("msg length not enough, should be: %d, get: %d", expect, reality)
}
