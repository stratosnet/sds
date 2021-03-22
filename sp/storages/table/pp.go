package table

import (
	"github.com/qsnetwork/qsds/utils/database"
	"time"
)

/*
CREATE TABLE `pp` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'Id of pp',
  `wallet_address` char(42) NOT NULL DEFAULT '' ,
  `network_address` varchar(32) NOT NULL DEFAULT '' ,
  `disk_size` bigint(20) unsigned NOT NULL DEFAULT '0' ,
  `free_disk` bigint(20) unsigned NOT NULL DEFAULT '0' ,
  `memory_size` bigint(20) unsigned NOT NULL DEFAULT '0' ,
  `os_and_ver` varchar(128) NOT NULL DEFAULT '' ,
  `cpu_info` varchar(64) NOT NULL DEFAULT '' ,
  `mac_address` varchar(17) NOT NULL DEFAULT '' ,
  `version` int(10) unsigned NOT NULL DEFAULT '0' ,
  `pub_key` varchar(1000) NOT NULL DEFAULT '' ,
  `state` tinyint(3) unsigned NOT NULL DEFAULT '0' COMMENT '0:offline,1:online',
  PRIMARY KEY (`id`),
  UNIQUE KEY `IDX_WALLET_ADDRESS` (`wallet_address`) USING HASH
) ENGINE=InnoDB  DEFAULT CHARSET=utf8;
 */

const (
	STATE_OFFLINE = 0
	STATE_ONLINE = 1
)

// PP 表
type PP struct {
	Id             uint32
	WalletAddress  string
	NetworkAddress string
	DiskSize       uint64
	FreeDisk       uint64
	MemorySize     uint64
	OsAndVer       string
	CpuInfo        string
	MacAddress     string
	Version        uint32
	PubKey         string
	State          byte
}

// TableName
func (p *PP) TableName() string {
	return "pp"
}

// PrimaryKey
func (p *PP) PrimaryKey() []string {
	return []string{"id"}
}

// SetData
func (p *PP) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(p, data)
}

// GetCacheKey
func (p *PP) GetCacheKey() string {
	return "pp#" + p.WalletAddress
}

// GetTimeOut
func (p *PP) GetTimeOut() time.Duration {
	return 0
}

// Where
func (p *PP) Where() map[string]interface{} {
	return map[string]interface{}{
		"where": map[string]interface{}{
			"wallet_address = ?": p.WalletAddress,
		},
	}
}

// Event
func (p *PP) Event(event int, dt *database.DataTable) {}
