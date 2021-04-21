package events

import (
	"context"
	"fmt"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto"
)

// RegisterNewPP P register to be PP
type RegisterNewPP struct {
	Server *net.Server
}

// GetServer
func (e *RegisterNewPP) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *RegisterNewPP) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *RegisterNewPP) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqRegisterNewPP)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqRegisterNewPP)

		rsp := &protos.RspRegisterNewPP{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
		}

		if !e.GetServer().Conf.Peers.RegisterSwitch {
			return rsp, header.RspRegisterNewPP
		}

		user := new(table.User)

		if ok, msg := e.Validate(body, user); !ok {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = msg
			return rsp, header.RspRegisterNewPP
		}

		newPP := &table.PP{
			WalletAddress:  body.WalletAddress,
			NetworkAddress: user.NetworkAddress,
			DiskSize:       body.DiskSize,
			MemorySize:     body.MemorySize,
			OsAndVer:       body.OsAndVer,
			CpuInfo:        body.CpuInfo,
			MacAddress:     body.MacAddress,
			Version:        body.Version,
			PubKey:         fmt.Sprintf("PubKeySecp256k1{%X}", body.PubKey),
			State:          table.STATE_OFFLINE,
		}

		if err := e.GetServer().CT.Save(newPP); err != nil {
			utils.ErrorLog(err)
		}

		return rsp, header.RspRegisterNewPP
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}

// Validate
func (e *RegisterNewPP) Validate(req *protos.ReqRegisterNewPP, user *table.User) (bool, string) {

	// todo change to read from redis
	pp := &table.PP{
		WalletAddress: req.WalletAddress,
	}
	if e.GetServer().CT.Fetch(pp) == nil {
		return false, "already PP, not register needed"
	}

	// check if register or not, todo change to read from redis
	user.WalletAddress = req.WalletAddress
	if e.GetServer().CT.Fetch(user) != nil {
		return false, "not register as PP, register first"
	}

	if len(req.PubKey) <= 0 || len(req.Sign) <= 0 {
		return false, "public key or signature is empty"
	}

	puk, err := crypto.UnmarshalPubkey(req.PubKey)
	if err != nil {
		return false, err.Error()
	}

	if !utils.ECCVerify([]byte(req.WalletAddress), req.Sign, puk) {
		return false, "signature verification failed"
	}

	return true, ""
}
