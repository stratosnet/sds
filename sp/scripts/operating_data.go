package scripts

import (
	"database/sql"
	"fmt"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/database"
	"github.com/stratosnet/sds/utils/database/drivers"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// OperatingData
type OperatingData struct {
	DT    *database.DataTable //
	Log   *utils.Logger
	Debug bool
	time  time.Time
}

// SetTime
func (c *OperatingData) SetTime(t time.Time) {
	c.time = t
}

// GetData
func (c *OperatingData) GetData() {

	logFile := "tmp/logs/operating_data.log"
	path, _ := filepath.Abs(filepath.Dir(logFile))
	if _, err := os.Stat(path); err != nil {
		os.MkdirAll(path, 0711)
	}
	c.Log = utils.NewLogger(logFile, false, true)
	c.Log.SetLogLevel(utils.Debug)

	var clientDownload int64
	var account int64
	var uploads int64
	var downloads int64
	var videoDownloads int64
	var share int64
	var albums int64

	var totalClientDownload int64
	var totalAccount int64
	var totalUploads int64
	var totalDownloads int64
	var totalVideoDownloads int64
	var totalShare int64
	var totalAlbums int64

	t := c.time
	endTime := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	startTime := endTime.AddDate(0, 0, -1)

	// count of client download in last day
	sql := "SELECT COUNT(*) clientDownload FROM client_download_record WHERE time >= ? AND time <= ?"
	row := c.Execute(sql, startTime.Unix(), endTime.Unix()-1)
	row.Scan(&clientDownload)

	// count of wallet creation in last day
	sql = "SELECT SUM(total) AS accounts FROM (SELECT COUNT(*) total FROM user WHERE register_time >= ? AND register_time <= ?) AS RESULT"
	row = c.Execute(sql, startTime.Unix(), endTime.Unix()-1)
	row.Scan(&account)

	// count of file upload in last day
	sql = "SELECT COUNT(*) AS uploads FROM file WHERE time >= ? AND time <= ?"
	row = c.Execute(sql, startTime.Unix(), endTime.Unix()-1)
	row.Scan(&uploads)

	// count of file download in last day
	sql = "SELECT COUNT(*) AS downloads FROM file_download WHERE time >= ? AND time <= ?"
	row = c.Execute(sql, startTime.Unix(), endTime.Unix()-1)
	row.Scan(&downloads)

	// count of video file download in last day
	videoList := []string{
		"lower(f.name) LIKE \"%.avi\"",
		"lower(f.name) LIKE \"%.wmv\"",
		"lower(f.name) LIKE \"%.mpeg\"",
		"lower(f.name) LIKE \"%.mp4\"",
		"lower(f.name) LIKE \"%.mkv\"",
		"lower(f.name) LIKE \"%.rmvb\"",
		"lower(f.name) LIKE \"%.rm\"",
	}
	sql = "SELECT COUNT(*) AS downloads FROM file_download d JOIN file f ON d.file_hash = f.hash WHERE d.time >= ? AND d.time <= ? AND ("
	sql = sql + strings.Join(videoList, " OR ") + ")"
	row = c.Execute(sql, startTime.Unix(), endTime.Unix()-1)
	row.Scan(&videoDownloads)

	// count of file sharing in last day
	sql = "SELECT COUNT(*) AS shares FROM user_share WHERE share_type = ? AND time >= ? AND time <= ?"
	row = c.Execute(sql, table.SHARE_TYPE_FILE, startTime.Unix(), endTime.Unix()-1)
	row.Scan(&share)

	// count of album in last day
	sql = "SELECT COUNT(*) AS albums FROM album WHERE time >= ? AND time <= ?"
	row = c.Execute(sql, startTime.Unix(), endTime.Unix()-1)
	row.Scan(&albums)

	// client download total
	sql = "SELECT COUNT(*) AS clientDownload FROM client_download_record"
	row = c.Execute(sql)
	row.Scan(&totalClientDownload)

	// wallet created total
	sql = "SELECT SUM(total) AS accounts FROM (SELECT COUNT(*) total FROM user) AS RESULT"
	row = c.Execute(sql)
	row.Scan(&totalAccount)

	// file uploaded total
	sql = "SELECT COUNT(*) AS uploads FROM file"
	row = c.Execute(sql)
	row.Scan(&totalUploads)

	// file downloaded total
	sql = "SELECT COUNT(*) AS downloads FROM file_download"
	row = c.Execute(sql)
	row.Scan(&totalDownloads)

	// video file downloaded total
	sql = "SELECT COUNT(*) AS downloads FROM file_download d JOIN file f ON d.file_hash = f.hash WHERE "
	sql = sql + strings.Join(videoList, " OR ")
	row = c.Execute(sql)
	row.Scan(&totalVideoDownloads)

	// file shared total
	sql = "SELECT COUNT(*) AS shares FROM user_share WHERE share_type = ?"
	row = c.Execute(sql, table.SHARE_TYPE_FILE)
	row.Scan(&totalShare)

	// album total
	sql = "SELECT COUNT(*) AS albums FROM album"
	row = c.Execute(sql)
	row.Scan(&totalAlbums)

	fmt.Println("date：", startTime.Format("2006/1/2"))
	fmt.Printf("client downloaded：%d\n", clientDownload)
	fmt.Printf("wallet created：%d\n", account)
	fmt.Printf("file uploaded：%d\n", uploads)
	fmt.Printf("file downloaded：%d\n", downloads)
	fmt.Printf("video downloaded：%d\n", videoDownloads)
	fmt.Printf("file shared：%d\n", share)
	fmt.Printf("album created：%d\n", albums)

	fmt.Println()
	fmt.Println("total：")
	fmt.Printf("client downloaded：%d\n", totalClientDownload)
	fmt.Printf("wallet created：%d\n", totalAccount)
	fmt.Printf("file uploaded：%d\n", totalUploads)
	fmt.Printf("file downloaded：%d\n", totalDownloads)
	fmt.Printf("video downloaded：%d\n", totalVideoDownloads)
	fmt.Printf("file shared：%d\n", totalShare)
	fmt.Printf("album created：%d\n", totalAlbums)
}

// Execute
func (c *OperatingData) Execute(sql string, args ...interface{}) *sql.Row {
	if c.Debug {
		c.Log.Log(utils.Debug, sql, args)
	}
	return c.DT.GetDriver().(*drivers.MySQL).GetDB().QueryRow(sql, args...)
}
