package bilibili

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-spider/common"
	"io"
	"net/http"
	"os"
)

func list() {
	videos, err := listAll()
	if err != nil {
		fmt.Println(err)
		return
	}
	r, err := json.Marshal(videos)
	if err != nil {
		fmt.Println(err)
		return
	}
	file, err := os.OpenFile("shuiqianxiaoxi.json", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = file.WriteString(string(r))
	if err != nil {
		fmt.Println(err)
	}
}

func listAll() (videos []Archive, err error) {
	pn := 1
	f := "https://api.bilibili.com/x/space/channel/video?mid=316568752&cid=171373&pn=%d&ps=30&order=0&ctype=0"
	for {
		fmt.Printf("get vidoe of page %d\n", pn)
		url := fmt.Sprintf(f, pn)
		r, err := request("GET", url, nil)
		if err != nil {
			return nil, err
		}
		v := VideoListResponse{}
		if err := json.Unmarshal(r, &v); err != nil {
			return nil, err
		}
		if len(v.Data.List.Archives) == 0 {
			fmt.Println("没有更多数据了。")
			break
		}
		videos = append(videos, v.Data.List.Archives...)
		pn++
	}
	return
}

// 请求
func request(method, url string, headers map[string]string) (r []byte, err error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return
	}
	defaultHeaders := map[string]string{"Accept": common.Accept, "user-agent": common.UserAgent}
	for k, v := range defaultHeaders {
		req.Header.Set(k, v)
	}
	if headers != nil && len(headers) > 0 {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("status code is not 200")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	return body, nil
}
