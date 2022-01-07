package serv

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/file"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/pp/types"
)

const (
	DefaultMsg = "Request Accepted"
)

type CmdResult struct {
	Msg string
}

type terminalCmd struct {
}

func TerminalAPI() *terminalCmd {
	return &terminalCmd{}
}

func (api *terminalCmd) Wallets(param []string) (CmdResult, error) {
	Wallets()
	return CmdResult{Msg: ""}, nil
}

func (api *terminalCmd) NewWallets(param []string) (CmdResult, error) {
	CreateWallet(param[0], param[1], param[2], param[3], param[4])
	return CmdResult{Msg: ""}, nil
}

func (api *terminalCmd) Login(param []string) (CmdResult, error) {
	err := Login(param[0], param[1])
	return CmdResult{Msg: ""}, err
}

func (api *terminalCmd) Start(param []string) (CmdResult, error) {
	peers.StartMining()
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) RegisterPP(param []string) (CmdResult, error) {
	event.RegisterNewPP()
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Activate(param []string) (CmdResult, error) {
	if len(param) != 3 {
		return CmdResult{Msg: ""}, errors.New("expecting 3 params. Input amount of tokens, fee amount and gas amount")
	}

	amount, err := strconv.ParseInt(param[0], 10, 64)
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid amount param. Should be an integer")
	}
	fee, err := strconv.ParseInt(param[1], 10, 64)
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid fee param. Should be an integer")
	}
	gas, err := strconv.ParseInt(param[2], 10, 64)
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid gas param. Should be an integer")
	}

	if setting.State != types.PP_INACTIVE {
		return CmdResult{Msg: "the pp is not inactive"}, nil
	}

	if !setting.IsPP {
		return CmdResult{Msg: "register as a PP node first"}, nil
	}

	if err := event.Activate(amount, fee, gas); err != nil {
		return CmdResult{Msg: ""}, err
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) UpdateStake(param []string) (CmdResult, error) {
	if len(param) != 4 {
		return CmdResult{Msg: ""}, errors.New("expecting 4 params. Input amount of stakeDelta, fee amount, " +
			"gas amount and flag of incrStake(0 for desc, 1 for incr)")
	}

	stakeDelta, err := strconv.ParseInt(param[0], 10, 64)
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid amount param. Should be an integer")
	}
	fee, err := strconv.ParseInt(param[1], 10, 64)
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid fee param. Should be an integer")
	}
	gas, err := strconv.ParseInt(param[2], 10, 64)
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid gas param. Should be an integer")
	}
	incrStake, err := strconv.ParseBool(param[3])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid flag for stake change. 0 for desc, 1 for incr")
	}

	if setting.State != types.PP_ACTIVE {
		fmt.Println("PP node not activated yet")
		return CmdResult{Msg: "PP node not activated yet"}, nil
	}

	if !setting.IsPP {
		return CmdResult{Msg: "register as a PP node first"}, nil
	}

	if err := event.UpdateStake(stakeDelta, fee, gas, incrStake); err != nil {
		return CmdResult{Msg: ""}, err
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Deactivate(param []string) (CmdResult, error) {
	if len(param) != 2 {
		return CmdResult{Msg: ""}, errors.New("expecting 2 params. Input fee amount and gas amount")
	}

	fee, err := strconv.ParseInt(param[0], 10, 64)
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid fee param. Should be an integert")
	}
	gas, err := strconv.ParseInt(param[1], 10, 64)
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid gas param. Should be an integer")
	}

	if setting.State == types.PP_INACTIVE {
		return CmdResult{Msg: "The node is already inactive"}, nil
	}

	if err := event.Deactivate(fee, gas); err != nil {
		return CmdResult{Msg: ""}, err
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Prepay(param []string) (CmdResult, error) {
	if len(param) != 3 {
		return CmdResult{Msg: ""}, errors.New("expecting 3 params. Input amount of tokens, fee amount and gas amount")
	}

	amount, err := strconv.ParseInt(param[0], 10, 64)
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid amount param. Should be an integer")
	}
	fee, err := strconv.ParseInt(param[1], 10, 64)
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid fee param. Should be an integer")
	}
	gas, err := strconv.ParseInt(param[2], 10, 64)
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("invalid gas param. Should be an integerr")
	}

	if err := event.Prepay(amount, fee, gas); err != nil {
		return CmdResult{Msg: ""}, err
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Upload(param []string) (CmdResult, error) {
	if len(param) == 0 {
		return CmdResult{}, errors.New("input upload file path")
	}
	isEncrypted := false
	if len(param) > 1 && param[1] == "encrypt" {
		isEncrypted = true
	}
	pathStr := file.EscapePath(param[0:1])
	event.RequestUploadFile(pathStr, "", isEncrypted, nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) UploadStream(param []string) (CmdResult, error) {
	if len(param) == 0 {
		return CmdResult{}, errors.New("input upload file path")
	}
	pathStr := file.EscapePath(param)
	event.RequestUploadStream(pathStr, "", nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) List(param []string) (CmdResult, error) {
	if len(param) == 0 {
		event.FindMyFileList("", event.NowDir, "", "", 0, true, nil)

	} else {
		event.FindMyFileList(param[0], event.NowDir, "", "", 0, true, nil)
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Download(param []string) (CmdResult, error) {
	if len(param) == 0 {
		return CmdResult{}, errors.New("input download path, e.g: sdm://account_address/file_hash|filename(optional)")
	}
	event.GetFileStorageInfo(param[0], "", "", false, false, nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) DeleteFn(param []string) (CmdResult, error) {
	if len(param) == 0 {
		fmt.Println("input file hash")
		return CmdResult{}, errors.New("input file hash")
	}
	if len(param[0]) != setting.FILEHASHLEN {
		return CmdResult{}, errors.New("input correct file hash")
	}
	event.DeleteFile(param[0], "", nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Ver(param []string) (CmdResult, error) {
	return CmdResult{Msg: fmt.Sprintf("version:", setting.Config.VersionShow)}, nil
}

func (api *terminalCmd) Monitor(param []string) (CmdResult, error) {
	ShowMonitor()
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) StopMonitor(param []string) (CmdResult, error) {
	StopMonitor()
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Config(param []string) (CmdResult, error) {
	if len(param) < 2 {
		return CmdResult{Msg: ""}, errors.New("input parameter name and value, 'name value' with space separator")
	}
	setting.SetConfig(param[0], param[1])

	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Mkdir(param []string) (CmdResult, error) {
	if len(param) == 0 {
		return CmdResult{Msg: ""}, errors.New("input directory name")
	}
	event.MakeDirectory(param[0], "", nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Rmdir(param []string) (CmdResult, error) {
	if len(param) == 0 {
		return CmdResult{Msg: ""}, errors.New("input directory name")
	}
	event.RemoveDirectory(param[0], "", nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Mvdir(param []string) (CmdResult, error) {
	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("input file hash and target directory path")
	}
	if len(param) == 1 { // root path
		event.MoveFileDirectory(param[0], event.NowDir, "", "", nil)
		return CmdResult{Msg: DefaultMsg}, nil
	}
	if event.NowDir == "" {
		event.MoveFileDirectory(param[0], "", param[1], "", nil)
	} else {
		event.MoveFileDirectory(param[0], event.NowDir, param[1], "", nil)
	}
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Cd(param []string) (CmdResult, error) {
	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("invalid param")
	}
	event.Goto(param[0])
	return CmdResult{Msg: ""}, nil
}

func (api *terminalCmd) SaveFile(param []string) (CmdResult, error) {
	if len(param) < 2 {
		return CmdResult{Msg: ""}, errors.New("input file hash and wallet address")
	}
	event.SaveOthersFile(param[0], param[1], "", nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) SharePath(param []string) (CmdResult, error) {
	if len(param) < 3 {
		return CmdResult{Msg: ""}, errors.New("input directory hash, expire time(0 for non-expire), is private (0:public,1:private)")
	}
	time, timeErr := strconv.Atoi(param[1])
	if timeErr != nil {
		return CmdResult{Msg: ""}, errors.New("input expire time(0 means non-expire)")
	}
	private, err := strconv.Atoi(param[2])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("input is_private (0:public,1:private)")
	}
	isPrivate := false
	if private == 1 {
		isPrivate = true
	}
	// if len(str1) == setting.FILEHASHLEN { //
	// 	event.GetReqShareFile("", str1, "", int64(time), isPrivate, nil)
	// } else {
	event.GetReqShareFile("", "", param[0], int64(time), isPrivate, nil)
	// }
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) ShareFile(param []string) (CmdResult, error) {
	if len(param) < 3 {
		return CmdResult{Msg: ""}, errors.New("input file hash or directory path, expire time(0 for non-expire), is private (0:public,1:private)")
	}
	time, timeErr := strconv.Atoi(param[1])
	if timeErr != nil {
		fmt.Println("input expire time(0 for non-expire)")
		return CmdResult{Msg: ""}, errors.New("input expire time(0 for non-expire)")
	}
	private, err := strconv.Atoi(param[2])
	if err != nil {
		return CmdResult{Msg: ""}, errors.New("input is private (0:public,1:private)")
	}
	isPrivate := false
	if private == 1 {
		isPrivate = true
	}
	event.GetReqShareFile("", param[0], "", int64(time), isPrivate, nil)
	// if len(str1) == setting.FILEHASHLEN { //
	// 	event.GetReqShareFile("", str1, "", int64(time), isPrivate, nil)
	// } else {
	// }
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) AllShare(param []string) (CmdResult, error) {
	event.GetAllShareLink("", nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) CancelShare(param []string) (CmdResult, error) {
	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("input share id")
	}
	event.DeleteShare(param[0], "", nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) GetShareFile(param []string) (CmdResult, error) {
	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("input share link and retrieval secret key(if any)")
	}
	if len(param) < 2 {
		event.GetShareFile(param[0], "", "", nil)
	} else {
		event.GetShareFile(param[0], param[1], "", nil)
	}

	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) PauseGet(param []string) (CmdResult, error) {
	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("input file hash of the pause")
	}
	event.DownloadSlicePause(param[0], "", nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) PausePut(param []string) (CmdResult, error) {
	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("input file hash of the pause")
	}
	event.UploadPause(param[0], "", nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) CancelGet(param []string) (CmdResult, error) {
	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("input file hash of the cancel")
	}
	event.DownloadSliceCancel(param[0], "", nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) CreateAlbum(param []string) (CmdResult, error) {
	if len(param) < 5 {
		return CmdResult{Msg: ""}, errors.New("input album name, album abstract, album cover hash, " +
			"album type(0:movie,1:music,2:other), file hash(if multiple file, separate by comma)")
	}
	files := make([]*protos.FileInfo, 0)
	strs := strings.Split(param[4], ",")
	for i, val := range strs {
		t := &protos.FileInfo{
			FileHash: val,
			SortId:   uint64(i),
		}
		files = append(files, t)
	}
	event.CreateAlbum(param[0], param[1], param[2], param[3], "", files, false, nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) AlbumList(param []string) (CmdResult, error) {
	event.FindMyAlbum("", 0, 0, "", "", nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) AlbumEdit(param []string) (CmdResult, error) {
	// if len(param) < 3 {
	// 	fmt.Println("input album id, action(0:add,1:delete,2:update), file hash(if multiple file, separated by comma)")
	// 	return false
	// }
	// if param[1] == "2" { // edit
	// 	event.EditAlbum(param[0], param[1], param[2], param[3], param[4], "", nil, nil)
	// } else { // add
	// 	strs := strings.Split(param[2], ",")
	// 	event.EditAlbum(param[0], param[1], "", "", "", "", strs, nil)
	// }
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) AlbumContent(param []string) (CmdResult, error) {
	if len(param) < 1 {
		fmt.Println("input album id")
		return CmdResult{Msg: ""}, errors.New("input album id")
	}
	event.AlbumContent(param[0], "", nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) AlbumSearch(param []string) (CmdResult, error) {
	if len(param) < 3 {
		return CmdResult{Msg: ""}, errors.New("input keyword, album type(0:movie,1:music,2:other), sort type(0:newest, 1:hottest)")
	}
	event.SearchAlbum(param[0], param[1], param[2], "", 0, 0, nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) invite(param []string) (CmdResult, error) {
	if len(param) < 1 {
		return CmdResult{Msg: ""}, errors.New("input invite code")
	}
	event.Invite(param[0], "", nil)
	return CmdResult{Msg: DefaultMsg}, nil
}

func (api *terminalCmd) Reward(param []string) (CmdResult, error) {
	event.GetReward("", nil)
	return CmdResult{Msg: DefaultMsg}, nil
}
