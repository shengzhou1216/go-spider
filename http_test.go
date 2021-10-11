package main

import (
	"testing"
	"net/http"
	"io"
)

func TestHttp(t *testing.T) {
	t.Run("test request header",func(t *testing.T) {
		url := "https://api.bilibili.com/x/space/channel/video?mid=316568752&cid=171373&pn=1&ps=30&order=0&ctype=0"
		req,err := http.NewRequest("GET", url,nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
		req.Header.Add("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.71 Safari/537.36")
		
		client := &http.Client{

		}
		resp , err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		t.Log("response:",resp.Body)
		t.Log("request:",resp.Request)

		body,err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("body:",string(body))
	})
}