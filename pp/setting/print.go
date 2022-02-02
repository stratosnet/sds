package setting

import (
	"github.com/stratosnet/sds/utils"
)

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
	utils.Log(str)
}
