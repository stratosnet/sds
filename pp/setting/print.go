package setting

import "fmt"

// ShowProgress
func ShowProgress(p float32) {
	f := int(p)
	m := int(100 - p)
	str := ""
	for i := 0; i < f; i++ {
		str += "#"
	}
	for i := 0; i < m; i++ {
		str += "-"
	}
	fmt.Println(str)
}
