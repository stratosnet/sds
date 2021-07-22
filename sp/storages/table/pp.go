package table

import (
	"github.com/stratosnet/sds/utils/database"
	"time"
)

/*
CREATE TABLE pp
(
    id              int unsigned     NOT NULL AUTO_INCREMENT COMMENT 'Id of pp' PRIMARY KEY,
    p2p_address     char(255)         NOT NULL DEFAULT '',
    wallet_address  char(42)         NOT NULL DEFAULT '',
    network_address varchar(32)      NOT NULL DEFAULT '',
    disk_size       bigint unsigned  NOT NULL DEFAULT '0',
    free_disk       bigint unsigned  NOT NULL DEFAULT '0',
    memory_size     bigint unsigned  NOT NULL DEFAULT '0',
    os_and_ver      varchar(128)     NOT NULL DEFAULT '',
    cpu_info        varchar(64)      NOT NULL DEFAULT '',
    mac_address     varchar(17)      NOT NULL DEFAULT '',
    version         int unsigned     NOT NULL DEFAULT '0',
    pub_key         varchar(1000)    NOT NULL DEFAULT '',
    state           tinyint unsigned NOT NULL DEFAULT '0' COMMENT '0:offline,1:online',
    active          tinyint          NOT NULL DEFAULT '0',
    UNIQUE KEY IDX_P2P_ADDRESS (p2p_address) USING HASH
) ENGINE = InnoDB
  DEFAULT CHARSET = UTF8MB4;
*/

const (
	STATE_OFFLINE = 0
	STATE_ONLINE  = 1
)

const (
	PP_INACTIVE = iota
	PP_ACTIVE
	PP_SUSPENDED
)

// PP table
type PP struct {
	Id             uint32
	P2pAddress     string
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
	Active         byte // Whether or not the PP is an active resource node on the stratos-chain
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
	return "pp#" + p.P2pAddress
}

// GetTimeOut
func (p *PP) GetTimeOut() time.Duration {
	return 0
}

// Where
func (p *PP) Where() map[string]interface{} {
	return map[string]interface{}{
		"where": map[string]interface{}{
			"p2p_address = ?": p.P2pAddress,
		},
	}
}

// Event
func (p *PP) Event(event int, dt *database.DataTable) {}
