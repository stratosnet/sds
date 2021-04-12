package api

import (
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/httpserv"
	"net/http"
)

func legal(w http.ResponseWriter, request *http.Request) {
	var (
		copyrightLegalName string
		yourLegalName      string
		company            string
		jobTitle           string
		email              string
		mailingAddress     string
		phone              string
		authority          string
		signature          string
	)

	data, err := HTTPRequest(request, w, true)
	if err != nil {
		return
	}
	if data["copyrightLegalName"] != nil {
		copyrightLegalName = data["copyrightLegalName"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "copyrightLegalName is required").ToBytes())
		return
	}
	if data["yourLegalName"] != nil {
		yourLegalName = data["yourLegalName"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "yourLegalName is required").ToBytes())
		return
	}
	if data["company"] != nil {
		company = data["company"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "company is required").ToBytes())
		return
	}
	if data["jobTitle"] != nil {
		jobTitle = data["jobTitle"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "jobTitle is required").ToBytes())
		return
	}
	if data["email"] != nil {
		email = data["email"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "email is required").ToBytes())
		return
	}
	if data["mailingAddress"] != nil {
		mailingAddress = data["mailingAddress"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "mailingAddress is required").ToBytes())
		return
	}
	if data["phone"] != nil {
		phone = data["phone"].(string)
	}
	if data["authority"] != nil {
		authority = data["authority"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "authority is required").ToBytes())
		return
	}
	if data["signature"] != nil {
		signature = data["signature"].(string)
	} else {
		w.Write(httpserv.NewJson(nil, setting.FAILCode, "signature is required").ToBytes())
		return
	}
	utils.DebugLog(copyrightLegalName, yourLegalName, company, jobTitle,
		email,
		mailingAddress,
		phone,
		authority,
		signature)
	w.Write(httpserv.NewJson(nil, setting.SUCCESSCode, "request success").ToBytes())
}
