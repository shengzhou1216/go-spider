package main

type BilibiliResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	ttl     int    `json:"ttl"`
}

type VideoListResponse struct {
	Data VideoList `json:"data"`
	BilibiliResponse
}

type VideoList struct {
	Page Page `json:"page"`
	List List `json:"list"`
}

type List struct {
	Cid      int       `json:"cid"`
	Count    int       `json:"count"`
	Cover    string    `json:"cover"`
	Intro    string       `json:"intro"`
	Mid      int       `json:"mid"`
	Mtime    int       `json:"mtime"`
	Name     string    `json:"name"`
	Archives []Archive `json:"Archives"`
}

type Archive struct {
	Aid       int    `json:"aid"`
	Bvid      string `json:"bvid"`
	Cid       int    `json:"cid"`
	Copyright int    `json:"copyright"`
	Ctime     int    `json:"ctime"`
	Desc      string `json:"desc"`
	Duration  int    `json:"duration"`
	Dynamic   string `json:"dynamic"`
	MissionId int `json:"mission_id"`
	Pic       string `json:"pic"`
	Pubdate   int    `json:"pubdate"`
	ShortLink string `json:"short_link"`
	Tid       int    `json:"tid"`
	Title     string `json:"title"`
	Tname     string `json:"tname"`
}

type Page struct {
	Count int `json:"count"`
	Num   int `json:"num"`
	Size  int `json:"size"`
}
