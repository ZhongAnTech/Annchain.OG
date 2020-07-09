package utilfuncs

import (
	"fmt"
	"os"
)

func PanicIfError(err error, message string) {
	if err != nil {
		fmt.Println("panic: " + message)
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
