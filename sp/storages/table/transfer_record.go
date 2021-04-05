package table

import (
	"github.com/qsnetwork/sds/utils/database"
)

/*

CREATE TABLE `transfer_record` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `file_slice_id` int(10) unsigned NOT NULL DEFAULT '0',
  `transfer_cer` char(64) NOT NULL DEFAULT '' ,
  `from_wallet_address` char(42) NOT NULL DEFAULT '' COMMENT 'origin PP wallet address',
  `to_wallet_address` char(42) NOT NULL DEFAULT '' COMMENT 'target PP wallet address',
  `from_network_address` varchar(32) NOT NULL DEFAULT '' COMMENT 'origin PP network address',
  `to_network_address` varchar(32) NOT NULL DEFAULT '' COMMENT 'target network address',
  `status` tinyint(3) unsigned NOT NULL DEFAULT '1' COMMENT '0:success,1:waiting,2:pending,3:error',
  `time` int(10) unsigned NOT NULL DEFAULT '0' COMMENT 'transfer finish time',
  PRIMARY KEY (`id`),
  KEY `IDX_FILE_SLICE_ID` (`file_slice_id`) USING BTREE,
  KEY `IDX_TRANSFER_CER` (`transfer_cer`) USING HASH
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8;

*/

const (

	TRANSFER_RECORD_STATUS_SUCCESS = 0

	TRANSFER_RECORD_STATUS_CHECK = 1

	TRANSFER_RECORD_STATUS_CONFIRM = 2

	TRANSFER_RECORD_STATUS_EXCEPTION = 3
)

// TransferRecord
type TransferRecord struct {
	Id                 uint32
	SliceHash          string
	TransferCer        string
	FromWalletAddress  string
	ToWalletAddress    string
	FromNetworkAddress string
	ToNetworkAddress   string
	Status             byte
	Time               int64
}

// TableName
func (t *TransferRecord) TableName() string {
	return "transfer_record"
}

// PrimaryKey
func (t *TransferRecord) PrimaryKey() []string {
	return []string{"id"}
}

// SetData
func (t *TransferRecord) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(t, data)
}

// GetCacheKey
func (t *TransferRecord) GetCacheKey() string {
	return "transfer_record#" + t.TransferCer
}

// Event
func (t *TransferRecord) Event(event int, dt *database.DataTable) {}
