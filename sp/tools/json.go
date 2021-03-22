package tools

import (
	"encoding/json"
	"github.com/qsnetwork/qsds/utils"
)

type JsonResult struct {
	Errcode int         `json:"errcode"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

func (jr *JsonResult) ToBytes() []byte {
	b, err := json.Marshal(jr)
	if err != nil {
		utils.ErrorLog(err.Error())
		return NewErrorJson(1001, "failed to marshal json").ToBytes()
	}
	return b
}

func NewErrorJson(errcode int, msg string) *JsonResult {
	return &JsonResult{
		Errcode: errcode,
		Message: msg,
	}
}

func NewJson(data interface{}, errcode int, msg string) *JsonResult {
	return &JsonResult{
		Errcode: errcode,
		Data:    data,
		Message: msg,
	}
}
