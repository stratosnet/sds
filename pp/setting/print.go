package setting

import (
	"context"

	"github.com/stratosnet/sds/pp"
)

// ShowProgress
func ShowProgressWithContext(ctx context.Context, p float32) {
	f := int(p)
	m := int(100 - p)
	str := ""
	for i := 0; i < f; i++ {
		str += "#"
	}
	for i := 0; i < m; i++ {
		str += "-"
	}
	pp.Log(ctx, str)
}
