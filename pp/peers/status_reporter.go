package peers

import (
	"time"

	"github.com/alex023/clock"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

func reportNodeStatusToSP() {
	utils.Log("report node status to SP")
	clock := clock.NewClock()
	clock.AddJobRepeat(time.Second*60, 1, reportNodeStatusToSP)
	if setting.IsStartMining {
		event.ReportNodeStatus()
	}
}
