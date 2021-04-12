package api

import (
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils/httpserv"
	"net/http"

	"github.com/google/uuid"
)

func createAlbum(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	albumName := ""
	albumBlurb := ""
	albumCoverHash := ""
	albumType := ""
	isPrivate := false
	var files []*protos.FileInfo
	if data["albumName"] != nil {
		albumName = data["albumName"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "albumName is required").ToBytes())
		return
	}

	if data["albumBlurb"] != nil {
		albumBlurb = data["albumBlurb"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "albumBlurb is required").ToBytes())
		return
	}

	if data["albumCoverHash"] != nil {
		albumCoverHash = data["albumCoverHash"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "albumCoverHash is required").ToBytes())
		return
	}

	if data["albumType"] != nil {
		albumType = data["albumType"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "albumType is required").ToBytes())
		return
	}

	if data["isPrivate"] != nil {
		if data["isPrivate"].(string) == "0" {
			isPrivate = false
		} else {
			isPrivate = true
		}
	}

	if data["files"] != nil {
		for _, val := range data["files"].([]interface{}) {
			m := val.(map[string]interface{})
			t := &protos.FileInfo{
				FileHash: m["fileHash"].(string),
				SortId:   uint64(m["id"].(float64)),
			}
			files = append(files, t)
		}
	}
	event.CreateAlbum(albumName, albumBlurb, albumCoverHash, albumType, uuid.New().String(), files, isPrivate, w)
}
