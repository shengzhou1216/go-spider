package spider

import (
	"errors"
	"golang.org/x/net/context"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type file struct {
	name string
}

type task struct {
	url  string
	file *file
}

type downloader interface {
	Start() error
	Stop() error
	Pause() error
	Execute(*task) error
}

type SimpleDownloader struct {
	ctx     context.Context
	tasks   chan *task
	cap     int
	timeout time.Duration
}

func (d *SimpleDownloader) Start() {
	go func() {
		for {
			select {
			case <-d.ctx.Done():
				log.Println("download done. exit.")
			}
		}
	}()
	select {
	case t := <-d.tasks:
		d.Execute(t)
	}
}

func (d *SimpleDownloader) Stop() {

}

func (d *SimpleDownloader) Pause() {

}

func (d *SimpleDownloader) Execute(t *task) error {
	client := http.Client{}
	req, err := http.NewRequest("GET", t.url, nil)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(d.ctx, d.timeout)
	defer cancel()
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
		return errors.New(resp.Status)
	}
	file, err := os.OpenFile(t.file.name, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, resp.Body)
	return err
}

func NewSimpleDownloader(cap int) *SimpleDownloader {
	taskCh := make(chan *task, cap)
	return &SimpleDownloader{
		tasks: taskCh,
		cap:   cap,
		ctx:   context.Background(),
	}
}
