package main

import (
	"fmt"
	"log"
	"spider"
)

func main() {
	simpleDownloader := spider.NewSimpleDownloader(100)
	for i := 0; i < 2; i++ {
		t := &spider.Task{
			File: &spider.File{
				Name: fmt.Sprintf("name-%d.tar.gz", i),
			},
			Url: "https://go.dev/dl/go1.18.1.linux-amd64.tar.gz",
		}
		simpleDownloader.AddTask(t)
	}
	err := simpleDownloader.Run()
	if err != nil {
		log.Printf("downloader run err: %v \n", err)
		return
	}
}
