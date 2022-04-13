package spider

import (
	"errors"
	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"
)

type File struct {
	Name string
}

type Task struct {
	Url  string
	File *File
}

type downloader interface {
	Run() error
	Stop() error
	Pause() error
	Execute(*Task) error
}

type SimpleDownloader struct {
	ctx         context.Context
	cancel      func()
	tasks       chan *Task
	cap         int
	taskTimeout time.Duration
	stopTimeout time.Duration
	sigs        []os.Signal
	// 停机中
	stopping bool
}

func (d *SimpleDownloader) Run() error {
	log.Println("run downloader...")
	s := make(chan os.Signal, 1)
	signal.Notify(s, d.sigs...)
	eg, ctx := errgroup.WithContext(d.ctx)
	// handle Task
	go func() {
		for {
			select {
			// 不断接收任务
			case t := <-d.tasks:
				go func(t *Task) {
					if err := d.Execute(t); err != nil {
						log.Printf("Task err: %s", err)
					}
				}(t)
			default:
				//log.Println("no more task.")
			}
		}
	}()
	// handle context done and cancel signal
	eg.Go(func() error {
		select {
		case <-d.ctx.Done():
			log.Println("context done.")
			return ctx.Err()
		case sig := <-s:
			log.Printf("accept exit signal: %s \n", sig)
			if err := d.Stop(); err != nil {
				log.Printf("failed to stop app: %v", err)
				return err
			}
		}
		return nil
	})

	log.Println("downloader started.")
	if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	log.Println("downloader exited successfully.")
	return nil
}

func (d *SimpleDownloader) Stop() error {
	// 1. 不在处理新的任务
	// 2. 等待所有任务完成（有一个超时设置）,超时后，关闭所有任务
	log.Println("wait downloader to stop...")
	// 停一下，等待子goroutine退出
	stopCtx, cancel := context.WithTimeout(context.Background(), d.stopTimeout)
	defer cancel()
	if d.cancel != nil {
		d.cancel()
	}
	select {
	case <-stopCtx.Done():
		log.Println("stop timeout. force quit.")
		// 这里还要监听服务是否全部退出了
	}
	return nil
}

func (d *SimpleDownloader) Pause() error {
	return nil
}

const (
	outputDir = "images"
)

func init() {
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		_ = os.MkdirAll(outputDir, 0755)
	}
}

func (d *SimpleDownloader) Execute(t *Task) error {
	log.Printf("execute download task: %v", t)
	client := http.Client{}
	req, err := http.NewRequest("GET", t.Url, nil)
	// Host: s.iimzt.com
	//User-Agent: Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:99.0) Gecko/20100101 Firefox/99.0
	//Accept: image/avif,image/webp,*/*
	//Accept-Language: en-US,en;q=0.5
	//Accept-Encoding: gzip, deflate, br
	//Connection: keep-alive
	//Referer: https://mmzztt.com/
	//Sec-Fetch-Dest: image
	//Sec-Fetch-Mode: no-cors
	//Sec-Fetch-Site: cross-site
	//If-Modified-Since: Wed, 23 Mar 2022 13:47:55 GMT
	//If-None-Match: "623b250b-29d3"
	//Cache-Control: max-age=0
	//TE: trailers
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:99.0) Gecko/20100101 Firefox/99.0")
	req.Header.Set("Accept", "image/avif,image/webp,*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Sec-Fetch-Dest", "image")
	req.Header.Set("Sec-Fetch-Mode", "no-cors")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("TE", "trailers")
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(d.ctx, d.taskTimeout)
	defer cancel()
	p := path.Join(outputDir, t.File.Name)

	// 监听context down
	go func() {
		select {
		case <-d.ctx.Done(): // parent context 取消
			log.Println("parent context done,cancel request.")
			// 移除下载的文件
			if err := os.Remove(p); err != nil {
				log.Println(err)
			}
			cancel()
		case <-ctx.Done(): // 请求context取消
			log.Printf("execute cancel. %v", ctx.Err())
		}
	}()
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if resp == nil {
		return errors.New("response is null")
	}
	defer resp.Body.Close()
	if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
		return errors.New(resp.Status)
	}
	file, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, resp.Body)
	return err
}

func (d *SimpleDownloader) AddTask(t *Task) error {
	d.tasks <- t
	return nil
}

func NewSimpleDownloader(cap int) *SimpleDownloader {
	taskCh := make(chan *Task, cap)
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)
	return &SimpleDownloader{
		tasks:       taskCh,
		cap:         cap,
		cancel:      cancelFunc,
		ctx:         ctx,
		sigs:        []os.Signal{syscall.SIGINT, syscall.SIGKILL, os.Interrupt},
		taskTimeout: 5 * time.Second,
		stopTimeout: 10 * time.Second,
	}
}
