package events

import (
	"context"
	"encoding/hex"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto"
)

// registerNewPP is a concrete implementation of event
// P register to be PP
type registerNewPP struct {
	event
}

const registerNewPPEvent = "register_new_pp"

// GetRegisterNewPPHandler creates event and return handler func for it
func GetRegisterNewPPHandler(s *net.Server) EventHandleFunc {
	return registerNewPP{
		newEvent(registerNewPPEvent, s, registerNewPPCallbackFunc),
	}.Handle
}

// registerNewPPCallbackFunc is the main process of register new PP
func registerNewPPCallbackFunc(_ context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {
	body := message.(*protos.ReqRegisterNewPP)

	rsp := &protos.RspRegisterNewPP{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
	}

	if s.Conf.Peers.RegisterSwitch {
		return rsp, header.RspRegisterNewPP
	}

	user := &table.User{}

	if ok, msg := validateNewPP(s, body, user); !ok {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = msg
		return rsp, header.RspRegisterNewPP
	}

	pp := &table.PP{
		WalletAddress:  body.WalletAddress,
		NetworkAddress: user.NetworkAddress,
		DiskSize:       body.DiskSize,
		MemorySize:     body.MemorySize,
		OsAndVer:       body.OsAndVer,
		CpuInfo:        body.CpuInfo,
		MacAddress:     body.MacAddress,
		Version:        body.Version,
		PubKey:         hex.EncodeToString(body.PubKey),
		State:          table.STATE_OFFLINE,
	}

	if err := s.CT.Save(pp); err != nil {
		utils.ErrorLog(err)
	}

	return rsp, header.RspRegisterNewPP
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *registerNewPP) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqRegisterNewPP{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}

// validateNewPP checks pp data
func validateNewPP(s *net.Server, req *protos.ReqRegisterNewPP, user *table.User) (bool, string) {

	// todo change to read from redis
	pp := &table.PP{
		WalletAddress: req.WalletAddress,
	}
	if s.CT.Fetch(pp) == nil {
		return false, "already PP, not register needed"
	}

	// check if register or not, todo change to read from redis
	user.WalletAddress = req.WalletAddress
	if s.CT.Fetch(user) != nil {
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
