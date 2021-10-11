package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"time"
)

type File struct {
	Type string
	Name string
	Size int
	Ext  string
	Mime string
}

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

// Downloader 下载器
type Downloader struct {
	*http.Client
	Tasks      []*DownloadTask
	Success    int // 成功数量
	Fail       int // 失败数量
	Processing int // 处理中数量
	Pending    int // 等待中数量
	StartAt    time.Time
	EndAt      time.Time
}

// NewDownloader 创建下载器
func NewDownloader() Downloader {
	return Downloader{
		Client: &http.Client{},
	}
}

// Start 启动
func (d *Downloader) Start() {
	done := make(chan int)
	defer close(done)
	d.StartAt = time.Now()
	log.Printf("开始执行任务，本次共有%d个任务\n", len(d.Tasks))
	for i, task := range d.Tasks {
		d.Processing++
		go d.execute(done, task)
		d.Tasks[i] = task
	}
	sum := len(d.Tasks)
	for sum > 0 {
		<-done
		sum--
		log.Printf("剩余任务数-%d\n", sum)
	}
	d.EndAt = time.Now()
}

// AddTask 添加任务
func (d *Downloader) AddTask(url, file string) (err error) {
	req, err := FormRequest(url, nil)
	if err != nil {
		return err
	}
	task := &DownloadTask{
		ID:      len(d.Tasks) + 1,
		Url:     url,
		Request: req,
		File: File{
			Name: file,
		},
	}
	d.Tasks = append(d.Tasks, task)
	return
}

func (d Downloader) checkError(err error, done chan int, task *DownloadTask) {
	task.Error = err
	d.Fail++
	d.Processing--
	done <- task.ID
}

// execute 执行任务
func (d *Downloader) execute(done chan int, task *DownloadTask) {
	task.Start = time.Now()
	defer func() {
		if msg := recover(); msg != nil {
			fmt.Println(msg, debug.Stack())
			d.checkError(errors.New(fmt.Sprintf("%v", msg)), done, task)
			return
		}
	}()
	resp, err := d.Client.Do(task.Request)
	if err != nil {
		d.checkError(err, done, task)
		return
	}
	defer resp.Body.Close()
	task.Response = resp
	file, err := os.OpenFile(task.File.Name, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		d.checkError(err, done, task)
		return
	}
	if resp.StatusCode != http.StatusOK {
		d.checkError(errors.New(resp.Status), done, task)
		return
	}
	s, err := io.Copy(file, resp.Body)
	if err != nil {
		d.checkError(errors.New(resp.Status), done, task)
		return
	}
	//size := resp.Header.Get("Content-Length")
	task.Size = int(s)
	task.End = time.Now()
	d.Processing--
	d.Success++
	done <- task.ID
}

func (d *Downloader) Result() {
	fmt.Println("任务总耗时:", d.EndAt.Sub(d.StartAt))
	fmt.Println("成功数量:", d.Success, " 失败数量：", d.Fail)
}
