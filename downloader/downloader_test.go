package downloader

import (
	"fmt"
	"testing"
)

func TestDownloader(t *testing.T) {
	t.Run("test download", func(t *testing.T) {
		downloader := NewDownloader()
		err := downloader.AddTask("http://tjg.gzhuibei.com/a/1/46416/1.jpg", "1.jpg")
		if err != nil {
			t.Fatal(err)
		}
		downloader.Start()
		t.Log("任务总耗时:", downloader.EndAt.Sub(downloader.StartAt))
		t.Log("成功数量:", downloader.Success, " 失败数量：", downloader.Fail)
		for _, task := range downloader.Tasks {
			t.Log("==============================================")
			t.Log("错误：", task.Error)
			t.Log("大小:", task.Size)
			t.Log("图片名称：", task.File.Name)
			t.Log("耗时:", task.End.Sub(task.Start))
		}
	})

	t.Run("test multi download", func(t *testing.T) {
		downloader := NewDownloader()
		for i := 1; i <= 100; i++ {
			//https://tjg.gzhuibei.com/a/1/46170/67.jpg
			err := downloader.AddTask(fmt.Sprintf("http://tjg.gzhuibei.com/a/1/46170/%d.jpg", i), fmt.Sprintf("%d.jpg", i))
			if err != nil {
				t.Fatal(err)
			}
		}
		downloader.Start()
		t.Log("任务总耗时:", downloader.EndAt.Sub(downloader.StartAt))
		t.Log("成功数量:", downloader.Success, " 失败数量：", downloader.Fail)
		for _, task := range downloader.Tasks {
			t.Log("==============================================")
			t.Log("错误：", task.Error)
			t.Log("大小:", task.Size)
			t.Log("图片名称：", task.File.Name)
			t.Log("耗时:", task.End.Sub(task.Start))
		}
	})

	t.Run("test progress", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			t.Log("#")
		}
	})
}
