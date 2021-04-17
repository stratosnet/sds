package events

import (
	"context"
	"encoding/hex"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/data"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
	"time"
)

// register is a concrete implementation of event
// register P and PP
type register struct {
	event
}

const registerEvent = "register"

// GetRegisterHandler creates event and return handler func for it
func GetRegisterHandler(server *net.Server) EventHandleFunc {
	return (&register{
		newEvent(registerEvent, server, registerCallbackFunc),
	}).Handle
}

// registerCallbackFunc is the main process of register
func registerCallbackFunc(ctx context.Context, s *net.Server, message proto.Message, _ spbf.WriteCloser) (proto.Message, string) {

	updateTips, forceUpdate := shouldUpdate(ctx, s)

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

	if body.Address.WalletAddress == "" || body.Address.NetworkAddress == "" {
		rsp.Result.Msg = "wallet address and net address can't be empty"
		rsp.Result.State = protos.ResultState_RES_FAIL
		return rsp, header.RspRegister
	}

	// check PP
	pp := &table.PP{WalletAddress: body.Address.WalletAddress}
	if s.CT.Fetch(pp) == nil {
		rsp.IsPP = true
	}

	// save user info
	user := &table.User{WalletAddress: body.Address.WalletAddress}

	isNewUser := false
	if s.CT.Fetch(user) != nil {
		s.UserCount++
		user.InvitationCode = utils.Get8BitUUID()
		user.BeInvited = 0
		user.RegisterTime = time.Now().Unix()
		user.Capacity = s.System.InitializeCapacity
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

	totalUsed, _ := s.CT.SumTable(new(table.File), "f.size", map[string]interface{}{
		"alias": "f",
		"join":  []string{"user_has_file", "f.hash = uhf.file_hash", "uhf"},
		"where": map[string]interface{}{"uhf.wallet_address = ?": user.WalletAddress},
	})

	user.UsedCapacity = uint64(totalUsed)

	if s.CT.Save(user) != nil {
		return rsp, header.RspRegister
	}

	if !isNewUser {
		return rsp, header.RspRegister
	}

	invite := &table.UserInvite{
		InvitationCode: user.InvitationCode,
		WalletAddress:  user.WalletAddress,
		Times:          0,
	}
	if err := s.CT.Save(invite); err != nil {
		utils.ErrorLogf(eventHandleErrorTemplate, "register", "update user invite to db", err)
	}

	return rsp, header.RspRegister
}

func shouldUpdate(ctx context.Context, s *net.Server) (updateTips bool, forceUpdate bool) {
	sys := new(data.System)

	if s.Load(sys) != nil {
		return
	}

	msgBuf := spbf.MessageFromContext(ctx)
	if msgBuf.MSGHead.Version >= sys.Version {
		return
	}

	updateTips = true

	if sys.ForceUpdate {
		forceUpdate = true
	}
	return
}

// Handle create a concrete proto message for this event, and handle the event asynchronously
func (e *register) Handle(ctx context.Context, conn spbf.WriteCloser) {
	go func() {
		target := &protos.ReqRegister{}
		if err := e.handle(ctx, conn, target); err != nil {
			utils.ErrorLog(err)
		}
	}()
}
