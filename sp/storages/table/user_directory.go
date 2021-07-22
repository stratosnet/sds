package table

import (
	"errors"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/database"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// DIRECTORY_SEP
const DIRECTORY_SEP = "/"

// UserDirectory
type UserDirectory struct {
	DirHash       string
	WalletAddress string
	Path          string
	Time          int64
}

// TableName
func (ud *UserDirectory) TableName() string {
	return "user_directory"
}

// PrimaryKey
func (ud *UserDirectory) PrimaryKey() []string {
	return []string{"dir_hash"}
}

// SetData
func (ud *UserDirectory) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(ud, data)
}

// GetCacheKey
func (ud *UserDirectory) GetCacheKey() string {
	return "user_directory#" + ud.DirHash
}

// GetTimeOut
func (ud *UserDirectory) GetTimeOut() time.Duration {
	return time.Second * 60
}

// Where
func (ud *UserDirectory) Where() map[string]interface{} {
	return map[string]interface{}{"where": map[string]interface{}{"dir_hash = ?": ud.DirHash}}
}

// Event
func (ud *UserDirectory) Event(event int, dt *database.DataTable) {}

// FindDirs
func (ud *UserDirectory) FindDirs(ct *database.CacheTable, walletAddress, directory string) []*protos.FileInfo {

	where := "wallet_address = ?"
	args := []interface{}{walletAddress}

	if directory != "" {
		where = where + " AND path REGEXP '^" + directory + "/[^/]+$'"
	} else {
		where = where + " AND path NOT REGEXP '[/]+'"
	}

	dirRes, err := ct.FetchTables([]UserDirectory{}, map[string]interface{}{
		"columns": "dir_hash, path, time",
		"where":   map[string]interface{}{where: args},
		"orderBy": "time DESC",
		//"cache":   map[string]interface{}{"lifeTime": time.Second * 1},
	})

	if err == nil {
		dirs := dirRes.([]UserDirectory)
		if len(dirs) > 0 {
			fileInfos := make([]*protos.FileInfo, len(dirs))
			for idx, dir := range dirs {
				sPath := ""
				if strings.ContainsRune(dir.Path, '/') {
					sPath = filepath.Dir(dir.Path)
				}
				fileInfos[idx] = &protos.FileInfo{
					FileName:           filepath.Base(dir.Path),
					FileSize:           0,
					FileHash:           dir.DirHash,
					CreateTime:         uint64(dir.Time),
					IsDirectory:        true,
					StoragePath:        sPath,
					OwnerWalletAddress: dir.WalletAddress,
				}
			}
			return fileInfos
		}
	}
	return []*protos.FileInfo{}
}

// FindFiles
func (ud *UserDirectory) FindFiles(ct *database.CacheTable, walletAddress, directory, fileName, fileHash, keyword string, orderType protos.FileSortType, asc bool) []*protos.FileInfo {

	where := "e.hash NOT IN (SELECT udmf.file_hash FROM user_directory AS ud join user_directory_map_file AS udmf ON ud.dir_hash = udmf.dir_hash AND ud.wallet_address = ? WHERE ud.path != \"\")"
	args := []interface{}{walletAddress, walletAddress}
	if directory != "" {
		where = "e.hash IN (SELECT udmf.file_hash FROM user_directory AS ud join user_directory_map_file AS udmf ON ud.dir_hash = udmf.dir_hash AND ud.wallet_address = ? WHERE ud.path = ?)"
		args = append(args, directory)
	}

	if fileName != "" {
		where = where + " AND e.name = ?"
		args = append(args, fileName)
	}

	if fileHash != "" {
		where = where + " AND e.hash = ?"
		args = append(args, fileHash)
	}

	if keyword != "" {
		where = where + " AND e.name LIKE \"%" + keyword + "%\""
	}

	params := map[string]interface{}{
		"alias":   "e",
		"columns": []string{"e.*, uhf.wallet_address"},
		"join":    []string{"user_has_file", "e.hash = uhf.file_hash AND uhf.wallet_address = ?", "uhf"},
		"where":   map[string]interface{}{where: args},
		//"cache":   map[string]interface{}{"lifeTime": time.Second * 60},
	}

	if orderType > protos.FileSortType_DEF {
		if orderType == protos.FileSortType_TIME {
			params["orderBy"] = "e.time"
		} else if orderType == protos.FileSortType_SIZE {
			params["orderBy"] = "e.size"
		} else {
			params["orderBy"] = "e.name"
		}
		sort := "DESC"
		if asc {
			sort = "ASC"
		}
		params["orderBy"] = params["orderBy"].(string) + " " + sort
	}

	res, err := ct.FetchTables([]File{}, params)

	if err == nil {
		files := res.([]File)
		if len(files) > 0 {
			fileInfos := make([]*protos.FileInfo, len(files))
			for idx, file := range files {
				fileInfos[idx] = &protos.FileInfo{
					FileName:           file.Name,
					FileSize:           file.Size,
					FileHash:           file.Hash,
					CreateTime:         uint64(file.Time),
					IsDirectory:        false,
					StoragePath:        directory,
					OwnerWalletAddress: walletAddress,
				}
			}

			return fileInfos
		}
	}
	return []*protos.FileInfo{}
}

// RecursFindDirs
func (ud *UserDirectory) RecursFindDirs(ct *database.CacheTable) []*protos.FileInfo {

	where := "wallet_address = ?"
	if ud.Path != "" {
		where = where + " AND path LIKE '" + ud.Path + "/%'"
	}
	res, err := ct.FetchTables([]UserDirectory{}, map[string]interface{}{
		"where": map[string]interface{}{where: ud.WalletAddress},
	})
	if err == nil {
		dirs := res.([]UserDirectory)
		if len(dirs) > 0 {
			fileInfos := make([]*protos.FileInfo, len(dirs))
			for idx, dir := range dirs {
				sPath := ""
				if strings.ContainsRune(dir.Path, '/') {
					sPath = filepath.Dir(dir.Path)
				}
				fileInfos[idx] = &protos.FileInfo{
					FileName:           filepath.Base(dir.Path),
					FileSize:           0,
					FileHash:           dir.DirHash,
					CreateTime:         uint64(dir.Time),
					IsDirectory:        true,
					StoragePath:        sPath,
					OwnerWalletAddress: dir.WalletAddress,
				}
			}
			return fileInfos
		}
	}
	return []*protos.FileInfo{}
}

// RecursFindFiles
func (ud *UserDirectory) RecursFindFiles(ct *database.CacheTable) []*protos.FileInfo {

	type FileEx struct {
		File
		Path string
	}
	res, err := ct.FetchTables([]FileEx{}, map[string]interface{}{
		"alias":   "f",
		"columns": "f.*, ud.path",
		"join": [][]string{
			{"user_has_file", "f.hash = uhf.file_hash AND uhf.wallet_address = ?", "uhf"},
			{"user_directory_map_file", "udmf.file_hash = uhf.file_hash AND uhf.wallet_address = udmf.owner_wallet", "udmf"},
			{"user_directory", "udmf.dir_hash = ud.dir_hash AND udmf.owner_wallet = ud.wallet_address AND ud.path LIKE '" + ud.Path + "%'", "ud"},
		},
		"where": map[string]interface{}{"": ud.WalletAddress},
	})

	if err == nil {
		files := res.([]FileEx)
		if len(files) > 0 {
			fileInfos := make([]*protos.FileInfo, len(files))
			for idx, file := range files {
				fileInfos[idx] = &protos.FileInfo{
					FileName:           file.Name,
					FileSize:           file.Size,
					FileHash:           file.Hash,
					CreateTime:         uint64(file.Time),
					IsDirectory:        false,
					StoragePath:        file.Path,
					OwnerWalletAddress: ud.WalletAddress,
				}
			}
			return fileInfos
		}
	}
	return []*protos.FileInfo{}
}

// DeleteFileMap
func (ud *UserDirectory) DeleteFileMap(ct *database.CacheTable) {
	ct.GetDriver().Delete("user_has_file", map[string]interface{}{
		"wallet_address = ? AND file_hash IN (select file_hash from user_directory_map_file where dir_hash = ?)": []interface{}{
			ud.WalletAddress, ud.DirHash,
		},
	})
	ct.GetDriver().Delete("user_directory_map_file", map[string]interface{}{
		"owner = ? AND dir_hash = ?": []interface{}{ud.WalletAddress, ud.DirHash},
	})
}

// GenericHash
func (ud *UserDirectory) GenericHash() string {
	if ud.WalletAddress != "" && ud.Path != "" {
		ud.DirHash = utils.CalcHash([]byte(ud.WalletAddress + ud.Path))
		return ud.DirHash
	}
	return ""
}

// OptPath optimize path
func (ud *UserDirectory) OptPath(directory string) (string, error) {

	if err := ud.ValidatePath(directory); err != nil {
		return "", err
	}

	pathsReady := strings.FieldsFunc(directory, func(r rune) bool {
		return r == '/'
	})
	var paths []string
	if len(pathsReady) > 0 {
		paths = make([]string, len(pathsReady))
		for idx, path := range pathsReady {
			paths[idx] = strings.TrimSpace(path)
		}
	}
	return strings.Join(paths, "/"), nil
}

// ValidatePath
func (ud *UserDirectory) ValidatePath(directory string) error {
	if directory == "" ||
		strings.TrimSpace(directory) == "" {
		return errors.New("directory path can't be empty")
	}
	dirLevels := strings.Split(directory, "/")
	if len(dirLevels) > 8 {
		return errors.New("directory level can be over 8")
	}
	if strings.HasSuffix(directory, "/") {
		return errors.New("directory name can't be empty")
	}
	if utf8.RuneCountInString(directory) > 512 {
		return errors.New("directory name is too long")
	}

	return nil
}
