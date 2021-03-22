package peers

import (
	"github.com/qsnetwork/qsds/pp/client"
	"github.com/qsnetwork/qsds/pp/event"
	"github.com/qsnetwork/qsds/pp/setting"
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
