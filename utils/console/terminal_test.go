package console

import (
	"fmt"
	"testing"
)

func testRun(t *testing.T) {
	f := func(line string, param []string) bool {
		fmt.Println(param)
		return true
	}

	f2 := func(line string, param []string) bool {
		fmt.Println(param)
		return true
	}
	Mystdin.RegisterProcessFunc("test", f, true)
	Mystdin.RegisterProcessFunc("test2", f2, true)
	Mystdin.Run()
}
