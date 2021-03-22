package events

import (
	"context"
	"github.com/qsnetwork/qsds/framework/spbf"
	"github.com/qsnetwork/qsds/msg/header"
	"github.com/qsnetwork/qsds/msg/protos"
	"github.com/qsnetwork/qsds/sp/net"
	"github.com/qsnetwork/qsds/sp/storages/table"
)

// AbstractAlbum
type AbstractAlbum struct {
	Server *net.Server
}

// GetServer
func (e *AbstractAlbum) GetServer() *net.Server {
	return e.Server
}

// SetServer
func (e *AbstractAlbum) SetServer(server *net.Server) {
	e.Server = server
}

// Handle
func (e *AbstractAlbum) Handle(ctx context.Context, conn spbf.WriteCloser) {

	target := new(protos.ReqAbstractAlbum)

	callback := func(message interface{}) (interface{}, string) {

		body := message.(*protos.ReqAbstractAlbum)

		rsp := &protos.RspAbstractAlbum{
			Result: &protos.Result{
				State: protos.ResultState_RES_SUCCESS,
			},
			WalletAddress:   body.WalletAddress,
			ReqId:           body.ReqId,
			MyAlbum:         nil,
			CollectionAlbum: nil,
		}

		if body.WalletAddress == "" {
			rsp.Result.State = protos.ResultState_RES_FAIL
			rsp.Result.Msg = "wallet address can't be empty"
			return rsp, header.RspSearchAlbum
		}

		rsp.MyAlbum = new(protos.AlbumNumber)
		rsp.CollectionAlbum = new(protos.AlbumNumber)

		type abstract struct {
			table.Album
			Type  byte
			Total int64
		}

		res, err := e.GetServer().CT.FetchTables([]abstract{}, map[string]interface{}{
			"columns": "type, COUNT(*) AS total",
			"where":   map[string]interface{}{"wallet_address = ?": body.WalletAddress},
			"groupBy": "type",
		})

		if err == nil {
			myAlbumAbstract := res.([]abstract)
			for _, ab := range myAlbumAbstract {
				rsp.MyAlbum.All = rsp.MyAlbum.All + ab.Total
				if ab.Type == byte(protos.AlbumType_VIDEO) {
					rsp.MyAlbum.Video = ab.Total
				} else if ab.Type == byte(protos.AlbumType_MUSIC) {
					rsp.MyAlbum.Music = ab.Total
				} else if ab.Type == byte(protos.AlbumType_OTHER) {
					rsp.MyAlbum.Other = ab.Total
				}
			}
		}

		res, err = e.GetServer().CT.FetchTables([]abstract{}, map[string]interface{}{
			"alias":   "a",
			"columns": "a.type, COUNT(*) AS total",
			"join":    []string{"user_collect_album", "uca.album_id = a.album_id AND uca.wallet_address = ?", "uca"},
			"where":   map[string]interface{}{"": body.WalletAddress},
			"groupBy": "type",
		})

		if err == nil {
			collectAlbumAbstract := res.([]abstract)
			for _, ab := range collectAlbumAbstract {
				rsp.CollectionAlbum.All = rsp.CollectionAlbum.All + ab.Total
				if ab.Type == byte(protos.AlbumType_VIDEO) {
					rsp.CollectionAlbum.Video = ab.Total
				} else if ab.Type == byte(protos.AlbumType_MUSIC) {
					rsp.CollectionAlbum.Music = ab.Total
				} else if ab.Type == byte(protos.AlbumType_OTHER) {
					rsp.CollectionAlbum.Other = ab.Total
				}
			}
		}

		return rsp, header.RspAbstractAlbum
	}

	go net.EventHandle(ctx, conn, target, callback, e.GetServer().Ver)
}
