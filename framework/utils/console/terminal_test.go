package console

import (
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	f := func(line string, param []string) bool {
		//fmt.Println(param)
		return true
	}

	f2 := func(line string, param []string) bool {
		//fmt.Println(param)
		return true
	}
	Mystdin.RegisterProcessFunc("test", f, true)
	Mystdin.RegisterProcessFunc("test2", f2, true)

	go Mystdin.Run()
	time.Sleep(time.Second) // Finish test after 1 second, otherwise Mystdin.Run() will last forever
}
