package events

import (
	"context"
	"encoding/hex"
	"github.com/qsnetwork/sds/framework/spbf"
	"github.com/qsnetwork/sds/msg/header"
	"github.com/qsnetwork/sds/msg/protos"
	"github.com/qsnetwork/sds/sp/net"
	"github.com/qsnetwork/sds/sp/storages/data"
	"github.com/qsnetwork/sds/sp/storages/table"
	"github.com/qsnetwork/sds/utils"
	"time"
)

// Register for P and PP
type Register struct {
	Server *net.Server
}

// GetServer
func (e *Register) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *Register) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *Register) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqRegister)

	updateTips := false
	forceUpdate := false
	sys := new(data.System)
	if e.GetServer().Load(sys) == nil {
		msgBuf := spbf.MessageFromContext(ctx)
		if msgBuf.MSGHead.Version < sys.Version {
			updateTips = true
			if sys.ForceUpdate {
				forceUpdate = true
			}
		}
	}

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqRegister)

		rsp := &protos.RspRegister{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			IsPP:          false,
			WalletAddress: body.Address.WalletAddress,
		}

		if updateTips {
			rsp.Result.Msg = rsp.Result.Msg + "client has newer version"
			if forceUpdate {
				rsp.Result.State = protos.ResultState_RES_FAIL
				rsp.Result.Msg = "client version is old, please update"
				return rsp, header.RspRegister
			}
		}

		if body.Address.WalletAddress == "" ||
			body.Address.NetworkAddress == "" {

			rsp.Result.Msg = "wallet address and net address can't be empty"
			rsp.Result.State = protos.ResultState_RES_FAIL
			return rsp, header.RspRegister
		}

		// check PP
		pp := &table.PP{WalletAddress: body.Address.WalletAddress}
		if e.GetServer().CT.Fetch(pp) == nil {
			rsp.IsPP = true
		}

		// save user info
		user := &table.User{WalletAddress: body.Address.WalletAddress}

		isNewUser := false
		if e.GetServer().CT.Fetch(user) != nil {
			e.GetServer().UserCount++
			user.InvitationCode = utils.Get8BitUUID()
			user.BeInvited = 0
			user.RegisterTime = time.Now().Unix()
			user.Capacity = e.GetServer().System.InitializeCapacity
			isNewUser = true
		}

		if rsp.IsPP {
			user.IsPp = 1
		}
		user.Belong = body.MyAddress.WalletAddress
		user.WalletAddress = body.Address.WalletAddress
		user.NetworkAddress = body.Address.NetworkAddress
		user.Puk = hex.EncodeToString(body.PublicKey)
		user.LastLoginTime = time.Now().Unix()
		user.LoginTimes = user.LoginTimes + 1

		totalUsed, _ := e.GetServer().CT.SumTable(new(table.File), "f.size", map[string]interface{}{
			"alias": "f",
			"join":  []string{"user_has_file", "f.hash = uhf.file_hash", "uhf"},
			"where": map[string]interface{}{"uhf.wallet_address = ?": user.WalletAddress},
		})

		user.UsedCapacity = uint64(totalUsed)
		if e.GetServer().CT.Save(user) == nil {
			if isNewUser {
				invite := &table.UserInvite{
					InvitationCode: user.InvitationCode,
					WalletAddress:  user.WalletAddress,
					Times:          0,
				}
				e.GetServer().CT.Save(invite)
			}
		}

		return rsp, header.RspRegister
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
