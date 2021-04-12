package table

import (
	"github.com/stratosnet/sds/utils/database"
	"time"
)

/*

CREATE TABLE `file_slice` (
  `id` int(11) NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `file_hash` char(64) NOT NULL DEFAULT '' COMMENT '文件hash',
  `slice_hash` char(64) NOT NULL DEFAULT '' COMMENT '文件切片hash',
  `slice_size` bigint(20) NOT NULL DEFAULT '0' COMMENT '切片大小',
  `slice_number` int(10) unsigned NOT NULL DEFAULT '1' COMMENT '切片号',
  `wallet_address` char(42) NOT NULL DEFAULT '' COMMENT '钱包地址',
  `network_address` varchar(32) NOT NULL DEFAULT '' COMMENT '网络地址',
  `status` tinyint(3) unsigned NOT NULL DEFAULT '0' COMMENT '状态：0成功，1待确认，2异常',
  `time` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '上传时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `IDX_SLICE_HASH` (`slice_hash`) USING HASH,
  KEY `IDX_WALLET_ADDRESS` (`wallet_address`) USING HASH,
  KEY `IDX_FILE_HASH` (`file_hash`) USING HASH
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8;

*/

const (

	// FILE_SLICE_STATUS_SUCCESS 成功
	FILE_SLICE_STATUS_SUCCESS = 0

	// FILE_SLICE_STATUS_CHECK 待确认
	FILE_SLICE_STATUS_CHECK = 1
)

// FileSlice
type FileSlice struct {
	Id               uint32
	FileHash         string
	SliceHash        string
	SliceSize        uint64
	SliceNumber      uint64
	SliceOffsetStart uint64
	SliceOffsetEnd   uint64
	Status           byte
	TaskId           string
	Time             int64
	FileSliceStorage
}

// TableName
func (f *FileSlice) TableName() string {
	return "file_slice"
}

// PrimaryKey
func (f *FileSlice) PrimaryKey() []string {
	return []string{"id"}
}

// SetData
func (f *FileSlice) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(f, data)
}

// GetCacheKey
func (f *FileSlice) GetCacheKey() string {
	if f.WalletAddress != "" {
		return "file_slice#" + f.SliceHash + "-" + f.WalletAddress
	}
	return "file_slice#" + f.SliceHash
}

// GetTimeOut
func (f *FileSlice) GetTimeOut() time.Duration {
	return time.Second * 60
}

// Where
func (f *FileSlice) Where() map[string]interface{} {
	where := map[string]interface{}{
		"where": map[string]interface{}{
			"slice_hash = ?": f.SliceHash,
		},
	}
	if f.WalletAddress != "" {
		where = map[string]interface{}{
			"alias":   "e",
			"columns": "e.*, fss.wallet_address, fss.network_address",
			"join": []string{
				"file_slice_storage", "e.slice_hash = fss.slice_hash", "fss",
			},
			"where": map[string]interface{}{
				"e.slice_hash = ? AND fss.wallet_address = ?": []interface{}{
					f.SliceHash, f.WalletAddress,
				},
			},
		}
	}
	return where
}

// Event
func (f *FileSlice) Event(event int, dt *database.DataTable) {
	switch event {
	case database.AFTER_INSERT:
		if f.WalletAddress != "" && f.SliceHash != "" {
			dt.StoreTable(&FileSliceStorage{SliceHash: f.SliceHash, WalletAddress: f.WalletAddress, NetworkAddress: f.NetworkAddress})
		}
	case database.BEFORE_DELETE:
		if f.WalletAddress != "" && f.SliceHash != "" {
			dt.DeleteTable(&FileSliceStorage{SliceHash: f.SliceHash, WalletAddress: f.WalletAddress, NetworkAddress: f.NetworkAddress})
		}
	}
}
