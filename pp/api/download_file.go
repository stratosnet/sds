package api

import (
	"net/http"

	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils/httpserv"
	"github.com/stratosnet/sds/utils/types"

	"github.com/google/uuid"
)

type downFile struct {
	FileHash           string `json:"fileHash"`
	OwnerWalletAddress string `json:"ownerWalletAddress"`
}

func downloadFile(w http.ResponseWriter, request *http.Request) {
	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	if data["fileHash"] == nil {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "fileHash is required").ToBytes())
		return
	}
	if data["ownerWalletAddress"] == nil {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "ownerWalletAddress is required").ToBytes())
		return
	}

	p := &downFile{
		FileHash:           data["fileHash"].(string),
		OwnerWalletAddress: data["ownerWalletAddress"].(string),
	}
	path := types.DataMashId{
		Owner: p.OwnerWalletAddress,
		Hash:  p.FileHash,
	}.String()
	downTaskID := uuid.New().String()

	event.GetFileStorageInfo(path, "", downTaskID, false, w)

	type df struct {
		TaskID             string `json:"taskID"`
		FileHash           string `json:"fileHash"`
		OwnerWalletAddress string `json:"ownerWalletAddress"`
	}
	tree := &df{
		TaskID:             downTaskID,
		FileHash:           p.FileHash,
		OwnerWalletAddress: p.OwnerWalletAddress,
	}
	result := make(map[string]*df, 0)
	result["downloadFile"] = tree
	_, _ = w.Write(httpserv.NewJson(result, setting.SUCCESSCode, "request success").ToBytes())
	return

}
