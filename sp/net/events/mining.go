package events

import (
	"context"
	"github.com/qsnetwork/sds/framework/spbf"
	"github.com/qsnetwork/sds/msg/header"
	"github.com/qsnetwork/sds/msg/protos"
	"github.com/qsnetwork/sds/sp/common"
	"github.com/qsnetwork/sds/sp/net"
	"github.com/qsnetwork/sds/sp/storages/table"
	"github.com/qsnetwork/sds/utils"
	"github.com/qsnetwork/sds/utils/crypto"
)

// Mining
type Mining struct {
	Server *net.Server
}

// GetServer
func (e *Mining) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *Mining) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *Mining) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqMining)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqMining)

		rsp := &protos.RspMining{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
		}


		if ok, msg := e.Validate(body); !ok {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = msg
			return rsp, header.RspMining
		}

		pp := &table.PP{WalletAddress: body.Address.WalletAddress}
		if e.GetServer().CT.Fetch(pp) != nil {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "not PP yet"
			return rsp, header.RspMining
		}

		// map net address with wallet address
		name := conn.(*spbf.ServerConn).GetName()
		e.GetServer().AddConn(name, body.Address.WalletAddress, conn.(*spbf.ServerConn))

		// send mining msg
		e.GetServer().HandleMsg(&common.MsgMining{
			WalletAddress:  body.Address.WalletAddress,
			NetworkAddress: body.Address.NetworkAddress,
			Name:           name,
			Puk:            body.PublicKey,
		})

		return rsp, header.RspMining
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}

// Validate
func (e *Mining) Validate(req *protos.ReqMining) (bool, string) {

	if req.Address.WalletAddress == "" ||
		req.Address.NetworkAddress == "" {
		return false, "wallet address or net address can't be empty"
	}

	if len(req.PublicKey) <= 0 {
		return false, "public key can't be empty"
	}

	if len(req.Sign) <= 0 {
		return false, "signature can't be empty"
	}

	puk, err := crypto.UnmarshalPubkey(req.PublicKey)
	if err != nil {
		return false, err.Error()
	}

	if !utils.ECCVerify([]byte(req.Address.WalletAddress), req.Sign, puk) {
		return false, "signature verification failed"
	}

	return true, ""

}
