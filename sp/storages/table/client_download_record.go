package table

import "github.com/qsnetwork/qsds/utils/database"

// ClientDownloadRecord
type ClientDownloadRecord struct {
	Id   uint64
	Type byte
	Time int64
}

// TableName
func (cdr *ClientDownloadRecord) TableName() string {
	return "client_download_record"
}

// PrimaryKey
func (cdr *ClientDownloadRecord) PrimaryKey() []string {
	return []string{"id"}
}

// SetData
func (cdr *ClientDownloadRecord) SetData(data map[string]interface{}) (bool, error) {
	return database.LoadTable(cdr, data)
}

// Event
func (cdr *ClientDownloadRecord) Event(event int, dt *database.DataTable) {}
