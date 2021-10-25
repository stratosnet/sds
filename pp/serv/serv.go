package serv

import (
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/utils"
)

func Start() {
	err := GetWalletAddress()
	if err != nil {
		utils.ErrorLog(err)
		return
	}
	peers.StartPP(event.RegisterEventHandle)

}
