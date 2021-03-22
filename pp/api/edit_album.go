package api

import (
	"github.com/qsnetwork/qsds/msg/protos"
	"github.com/qsnetwork/qsds/pp/event"
	"github.com/qsnetwork/qsds/pp/setting"
	"github.com/qsnetwork/qsds/utils/httpserv"
	"net/http"

	"github.com/google/uuid"
)

func editAlbum(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	albumID := ""
	albumCoverHash := ""
	albumName := ""
	albumBlurb := ""
	isPrivate := false
	var changeFiles []*protos.FileInfo

	if data["albumID"] != nil {
		albumID = data["albumID"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "albumID is required").ToBytes())
		return
	}
	if data["isPrivate"] != nil {
		if data["isPrivate"].(string) == "0" {
			isPrivate = false
		} else {
			isPrivate = true
		}
	}
	if data["changeFiles"] != nil {
		for _, f := range data["changeFiles"].([]interface{}) {
			m := f.(map[string]interface{})
			t := &protos.FileInfo{
				FileHash: m["fileHash"].(string),
				SortId:   uint64(m["id"].(float64)),
			}
			changeFiles = append(changeFiles, t)
		}
	}
	if data["albumCoverHash"] != nil {
		albumCoverHash = data["albumCoverHash"].(string)
	}
	if data["albumName"] != nil {
		albumName = data["albumName"].(string)
	}
	if data["albumBlurb"] != nil {
		albumBlurb = data["albumBlurb"].(string)
	}

	event.EditAlbum(albumID, albumCoverHash, albumName, albumBlurb, uuid.New().String(), changeFiles, isPrivate, w)
}
