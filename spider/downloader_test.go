package spider

import "testing"

var simpleDownloader downloader

func init() {
	simpleDownloader = NewSimpleDownloader(100)
}

func TestSimpleDownloader_Start(t *testing.T) {
	err := simpleDownloader.Run()
	if err != nil {
		t.Error(err)
		return
	}
}
