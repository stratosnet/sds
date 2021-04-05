package peers

import (
	"github.com/qsnetwork/sds/pp/client"
	"github.com/qsnetwork/sds/pp/event"
	"github.com/qsnetwork/sds/pp/setting"
)

// InitPPList
func initPPList() {
	pplist := setting.GetLocalPPList()
	if len(pplist) == 0 {
		event.GetPPList()
	} else {
		for _, ppInfo := range pplist {
			client.PPConn = client.NewClient(ppInfo.NetworkAddress, true)
			if client.PPConn == nil {

				setting.DeletePPList(ppInfo.NetworkAddress)
			} else {
				event.RegisterChain(false)
				return
			}
		}

		event.GetPPList()
	}
}

func initBPList() {
	if !setting.InitBPList() {
		event.GetBPList()
	}
}
