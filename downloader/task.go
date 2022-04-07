package downloader
import (
	"time"
	"net/http"
)
// DownloadTask 下载任务
type DownloadTask struct {
	ID    int
	Start time.Time
	End   time.Time
	Url   string
	File
	Request  *http.Request
	Response *http.Response
	Error    error
}