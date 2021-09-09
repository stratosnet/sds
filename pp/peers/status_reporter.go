package peers

import (
	"time"

	"github.com/alex023/clock"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/utils"
)

func reportNodeStatusToSP() {
	utils.Log("report node status to SP")
	clock := clock.NewClock()
	clock.AddJobRepeat(time.Second*60, 0, event.ReportNodeStatus)
}
