package downloader

import (
	"context"
	"errors"
	"fmt"
	"go-spider/common"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"runtime/pprof"
	"sync"
	"syscall"
	"time"
)

// Downloader 下载器
type Downloader struct {
	*http.Client
	Tasks        []*DownloadTask
	Success      int // 成功数量
	Fail         int // 失败数量
	Processing   int // 处理中数量
	Pending      int // 等待中数量
	Finished     int // 结束的数量
	StartAt      time.Time
	EndAt        time.Time
	DownloadSize float64
	ctx          context.Context
	cancel       func()
	sigs         []os.Signal // 信号
	wg           *sync.WaitGroup
}

// NewDownloader 创建下载器
func NewDownloader() Downloader {
	ctx, cancel := context.WithCancel(context.Background())
	return Downloader{
		Client: &http.Client{},
		ctx:    ctx,
		cancel: cancel,
		sigs:   []os.Signal{os.Interrupt, syscall.SIGINT, syscall.SIGKILL},
		wg:     &sync.WaitGroup{},
	}
}

func (d *Downloader) setupPprof(fn func()) {
	file, _ := os.OpenFile("cpu.pprof", os.O_CREATE|os.O_RDWR, 0644)
	defer file.Close()
	pprof.StartCPUProfile(file)
	defer pprof.StopCPUProfile()
	fn()
}

// 带性能监控
func (d *Downloader) StartWitiPprof() {
	d.setupPprof(d.Start)
}

// Start 启动
func (d *Downloader) Start() {
	// file, _ := os.OpenFile("cpu.pprof", os.O_CREATE|os.O_RDWR, 0644)
	// defer file.Close()
	// pprof.StartCPUProfile(file)
	// defer pprof.StopCPUProfile()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, d.sigs...)
	go func() {
		select {
		case <-sig:
			d.cancel()
		case <-d.ctx.Done():
			d.cancel()
		}
	}()
	d.StartAt = time.Now()
	log.Printf("开始执行任务，本次共有%d个任务\n", len(d.Tasks))

	for i, task := range d.Tasks {
		d.Processing++
		d.wg.Add(1)
		go func() {
			d.execute(task)
		}()
		d.Tasks[i] = task
	}
	d.wg.Wait()
	// TODO: 任务越多，执行到后面越慢。最后一个任务一一直无法结束，一致等待。
	d.EndAt = time.Now()
}

// AddTask 添加任务
func (d *Downloader) AddTask(url, file string) (err error) {
	req, err := common.FormRequest(url, nil)
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

// func (d *Downloader) checkError(err error, done chan int, task *DownloadTask) {
// 	task.Error = err
// 	// d.Fail++
// 	// d.Processing--
// 	done <- task.ID
// }

// execute 执行下载任务
func (d Downloader) execute(task *DownloadTask) error {
	log.Printf("完成进度:%d/%d\n", d.Finished, len(d.Tasks))
	task.Start = time.Now()
	defer func() {
		d.wg.Done()
		d.Finished++
		if msg := recover(); msg != nil {
			log.Println(msg, debug.Stack())
			task.Error = errors.New(fmt.Sprintf("%v", msg))
			return
		}
	}()
	task.Request.WithContext(d.ctx)
	resp, err := d.Client.Do(task.Request)
	if err != nil {
		task.Error = err
		return err
	}
	defer resp.Body.Close()
	task.Response = resp
	file, err := os.OpenFile(task.File.Name, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		task.Error = err
		return err
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
		task.Error = errors.New(resp.Status)
		return errors.New(resp.Status)
	}
	s, err := io.Copy(file, resp.Body)
	if err != nil {
		task.Error = errors.New(resp.Status)
		return errors.New(resp.Status)
	}
	//size := resp.Header.Get("Content-Length")
	task.Size = float64(s)
	// d.DownloadSize += task.Size
	task.End = time.Now()
	// d.Processing--
	// d.Success++
	return nil
}

const statisticFile = "statistic.md"

func (d *Downloader) Result() {
	taskCount := len(d.Tasks)
	timeConsumption := d.EndAt.Sub(d.StartAt)
	for _, t := range d.Tasks {
		if t.Error != nil {
			d.Fail++
		} else {
			d.DownloadSize += t.Size
			d.Success++
		}
	}
	downloadSpeed := float64(int(d.DownloadSize)>>20) / (d.EndAt.Sub(d.StartAt).Seconds())
	taskPerSec := float64(d.Success) / (d.EndAt.Sub(d.StartAt).Seconds())
	log.Println("任务总数：", taskCount, "成功数量:", d.Success, " 失败数量：", d.Fail, "任务总耗时:", d.EndAt.Sub(d.StartAt),
		"平均每秒完成任务数为:", taskPerSec, "下载速度(M/s):", downloadSpeed)
	if _, err := os.Stat(statisticFile); os.IsNotExist(err) {
		file, err := os.Create(statisticFile)
		if err != nil {
			log.Fatal(err)
		}
		_, err = file.WriteString(fmt.Sprintf("| 任务总数 | 成功数量 | 失败数量 | 总耗时 | 平均每秒完成任务数 | 下载速度 |\n| ------ | ------ | ------ | ------ | ------ | ------ |\n"))
		if err != nil {
			log.Fatal(err)
		}
		file.Close()
	}
	file, err := os.OpenFile(statisticFile, os.O_APPEND|os.O_WRONLY, 0666)
	defer file.Close()
	if err != nil {
		log.Fatal(err)
	}
	_, err = file.WriteString(fmt.Sprintf("| %d | %d | %d | %s | %f | %f\n", len(d.Tasks), d.Success, d.Fail, timeConsumption, taskPerSec, downloadSpeed))
	if err != nil {
		log.Fatal(err)
	}
	for _, task := range d.Tasks {
		if task.Error != nil {
			log.Println("error:", task.Error)
		}
	}
}
