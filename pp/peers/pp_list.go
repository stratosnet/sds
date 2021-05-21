package peers

import (
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
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
