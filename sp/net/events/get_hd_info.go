package events

import (
	"context"
	"github.com/stratosnet/sds/framework/spbf"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/net"
	"github.com/stratosnet/sds/sp/storages/table"
)

type GetHDInfo struct {
	Server *net.Server
}

// GetServer
func (e *GetHDInfo) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *GetHDInfo) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *GetHDInfo) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.RspGetHDInfo)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.RspGetHDInfo)

		user := &table.User{WalletAddress: body.WalletAddress}
		if e.GetServer().CT.Fetch(user) == nil {

			user.FreeDisk = body.DiskFree
			user.DiskSize = body.DiskSize
			e.GetServer().CT.Save(user)

			if user.IsPp == 1 {

				pp := &table.PP{WalletAddress: body.WalletAddress}
				if e.GetServer().CT.Fetch(pp) == nil {
					pp.DiskSize = body.DiskSize
					pp.FreeDisk = body.DiskFree
					e.GetServer().CT.Save(pp)

					// check if reported disk size is within configured scale
					if float32(body.DiskSize-body.DiskFree)/float32(body.DiskSize) > e.GetServer().Conf.Peers.ProvideDiskScale {

						// if oversized, removed from hashring, don't assign task anymore
						e.GetServer().HashRing.SetOffline(body.WalletAddress)

						// todo: disk clean or file re-allocation?
					}
				}
			}
		}

		return nil, ""
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
