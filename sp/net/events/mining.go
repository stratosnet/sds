package events

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/sp/common"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
)

// mining is a concrete implementation of event
type mining struct {
	event
}

const miningEvent = "mining"

// GetMiningHandler creates event and return handler func for it
func GetMiningHandler(s *net.Server) EventHandleFunc {
	e := mining{newEvent(miningEvent, s, miningCallbackFunc)}
	return e.Handle
}

// miningCallbackFunc is the main process of mining
func miningCallbackFunc(_ context.Context, s *net.Server, message proto.Message, conn spbf.WriteCloser) (proto.Message, string) {

	body := message.(*protos.ReqMining)

	rsp := &protos.RspMining{
		Result: &protos.Result{
			State: protos.ResultState_RES_SUCCESS,
		},
	}

	if ok, msg := validateMiningRequest(body); !ok {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = msg
		return rsp, header.RspMining
	}

	pp := &table.PP{WalletAddress: body.Address.WalletAddress}
	if s.CT.Fetch(pp) != nil {
		rsp.Result.State = protos.ResultState_RES_FAIL
		rsp.Result.Msg = "not PP yet"
		return rsp, header.RspMining
	}

	// map net address with wallet address
	name := conn.(*spbf.ServerConn).GetName()
	s.AddConn(name, body.Address.WalletAddress, conn)

	// send mining msg
	s.HandleMsg(&common.MsgMining{
		WalletAddress: body.Address.WalletAddress,
		NetworkId:     setting.ToString(body.Address.NetworkId),
		Name:          name,
	})

	return rsp, header.RspMining
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *mining) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqMining{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}

// validateMiningRequest checks requests parameters
func validateMiningRequest(req *protos.ReqMining) (bool, string) {
	if req.Address.WalletAddress == "" || req.Address.NetworkId.NetworkAddress == "" {
		return false, "wallet address or net address can't be empty"
	}

	if req.Address.NetworkId.PublicKey == "" {
		return false, "public key can't be empty"
	}

	if len(req.Sign) <= 0 {
		return false, "signature can't be empty"
	}

	if !utils.ECCVerifyString([]byte(req.Address.WalletAddress), req.Sign, req.Address.NetworkId.PublicKey) {
		return false, "signature verification failed"
	}

	return true, ""
}
