package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

const (
	baseUrl                 = "https://www.tujidao.com"
	albumImageBaseUrlFormat = "https://tjg.gzhuibei.com/a/1/%d/%d.jpg"
	cookie                  = "UM_distinctid=17c693cc8ee4fe-0cee9e41dd4ec6-b7a1b38-144000-17c693cc8ef49b; PHPSESSID=3vioa1ieltdv1t0kje0meduuko; uid=229195; name=asdf0823; leixing=3; CNZZDATA1257039673=314412289-1633858333-%7C1633872601"
	imagesBaseDir           = "images"
)

type Tag struct {
	Name  string
	Url   string
	Pages int // 总页数
}

// Album 相册
type Album struct {
	Title        string
	Url          string
	Count        int // 图片数量
	Tag          Tag
	Id           int
	User         User
	Organization Organization
}

// User 人物
type User struct {
	Name string
	Url  string
}

// Organization 机构
type Organization struct {
	Name string
	Url  string
}

type Category struct {
	Name string
	Url  string
}

// 选择标签
func chooseTag(tags []Tag) Tag {
	log.Println("可选标签如下: ")
	for ix, tag := range tags {
		log.Printf("(%d)%s", ix+1, tag.Name)
	}
	tagIndex := 0
	log.Println("请选择一个tag(序号): ")
	for {
		_, err := fmt.Scanln(&tagIndex)
		if err != nil {
			log.Println(err)
			continue
		}
		if tagIndex < 1 || tagIndex > len(tags) {
			log.Println("超出范围了，请重新选择标签。")
			continue
		}
		break
	}
	return tags[tagIndex-1]
}

func choosePage(pages int) int {
	var page int
	for {
		log.Println("请输入要查看的页码：")
		if _, err := fmt.Scanln(&page); err != nil {
			log.Println("无效页码！请重新输入")
			continue
		}
		if page < 1 || page > pages {
			log.Printf("请输入1-%d范围内的数字", pages)
			continue
		}
		break
	}
	log.Printf("第%d页中的相册如下: ", page)
	return page
}

func TujidaoSpiderRun() {
	client := &http.Client{}
	tags, _ := getTagsAndCategories(client)
	for {
		// 选择tag
		tag := chooseTag(tags)
		log.Printf("你选择的标签是:%s", tag.Name)
		// 获取tag下的页数
		pages := tag.getPages(client)
		if pages == 0 {
			log.Println("此标签中没有数据，请重新选择标签")
			continue
		}
		log.Printf("此标签下共有%d页\n", pages)
		// 选择page
		page := choosePage(pages)
	AfterChoosePage:
		log.Printf("第%d页中的相册如下: ", page)
		// 列出相册
		albums := tag.listAlbums(client, tag.PageUrl(page))
		for ix, a := range albums {
			log.Printf("(%d)%s(%d)", ix+1, a.Title, a.Count)
		}
	ChoosePageOrDownload:
		log.Println("下载此页请输入D.要下载指定页请输入DP，其中P为页码；如D1，表示下载第1页。输入数字可切换页码")
		var cmd string
		var downloadPage int
		var isDownload bool
		if _, err := fmt.Scanln(&cmd); err != nil {
			log.Println(err)
			continue
		}
		re := regexp.MustCompile(`D(\d*)`)
		if re.Match([]byte(cmd)) {
			// 下载
			matchs := re.FindSubmatch([]byte(cmd))
			sp := matchs[1]
			if len(sp) == 0 {
				// 下载当前页
				downloadPage = page
			} else {
				if p, err := strconv.ParseInt(string(sp), 10, 0); err != nil {
					log.Println(err)
					goto ChoosePageOrDownload
				} else {
					if p < 1 || int(p) > pages {
						log.Println("无效页码，请重新输入")
						goto ChoosePageOrDownload
					} else {
						// download
						downloadPage = int(p)
					}
				}
			}

			isDownload = true
		} else {
			// 页码
			if p, err := strconv.ParseInt(cmd, 10, 0); err != nil {
				log.Println(err)
				goto ChoosePageOrDownload
			} else {
				page = int(p)
				goto AfterChoosePage
			}
		}
		if isDownload {
			// 初始化下载器
			downloader := NewDownloader()
			downloadAlbums := tag.listAlbums(client, tag.PageUrl(downloadPage))
			// 添加任务
			for _, a := range downloadAlbums {
				if err := AddAlbumTask(&downloader, &a); err != nil {
					log.Println(err)
					continue
				}
			}
			// 下载相册
			downloader.Start()
			downloader.Result()
		}
	}
}

// AddAlbumTask 将相册添加到任务中
func AddAlbumTask(downloader *Downloader, album *Album) (err error) {
	dir, err := album.LocalDir()
	if err != nil {
		return
	}
	for i := 1; i <= album.Count; i++ {
		img := fmt.Sprintf("%d.jpg", i)
		err = downloader.AddTask(fmt.Sprintf(albumImageBaseUrlFormat, album.Id, i), path.Join(dir, img))
		if err != nil {
			return
		}
	}
	return
}

// 获取tag下相册的页数
func (t *Tag) getPages(client *http.Client) int {
	// 第一页
	headers := map[string]string{"cookie": cookie}
	doc := requestDocument(client, t.Url, headers)
	// 页数
	var pages int
	if href, exists := doc.Find("#pages a").Last().Attr("href"); exists {
		re := regexp.MustCompile(`page=(\d+)`)
		matchs := re.FindSubmatch([]byte(href))
		if r, err := strconv.ParseInt(string(matchs[1]), 0, 10); err == nil {
			pages = int(r)
		} else {
			log.Fatal(err)
		}
	}
	t.Pages = pages
	return pages
}

// 列出tag下的相册
func (t *Tag) listAlbums(client *http.Client, url string) (albums []Album) {
	headers := map[string]string{"cookie": cookie}
	doc := requestDocument(client, url, headers)
	doc.Find(".hezi ul li").Each(func(i int, li *goquery.Selection) {
		if id, exists := li.Attr("id"); exists {
			idd, err := strconv.ParseInt(id, 10, 0)
			if err != nil {
				log.Println(err)
				return
			}
			// 相册图片数
			text := li.Find(".shuliang").Text()
			text = strings.ReplaceAll(text, "P", "")
			text = strings.ReplaceAll(text, "p", "")
			count, err := strconv.ParseInt(text, 10, 0)
			if err != nil {
				log.Fatal(err)
			}

			album := Album{
				Id:    int(idd),
				Count: int(count),
			}

			li.Find("p").Each(func(j int, p *goquery.Selection) {
				name := p.Find("a").Text()
				url, _ := p.Find("a").Attr("href")
				switch j {
				case 0:
					album.Organization = Organization{name, url}
				case 1:
					album.Tag = Tag{Name: name, Url: url}
				case 2:
					album.User = User{name, url}
				case 3:
					album.Title = strings.ReplaceAll(name, " ", "")
					album.Url = url
				}
			})

			albums = append(albums, album)
		}
	})
	return
}

// 请求html，返回document对象
func requestDocument(client *http.Client, url string, headers map[string]string) (doc *goquery.Document) {
	if !strings.HasPrefix(url, "http") {
		url = baseUrl + url
	}
	req, err := FormRequest(url, headers)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("status code error:%d %s", resp.StatusCode, resp.Status)
	}
	doc, err = goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	//log.Println(doc.Html())
	return
}

// 获取tag和category
func getTagsAndCategories(client *http.Client) (tags []Tag, categories []Category) {
	doc := requestDocument(client, baseUrl, nil)
	doc.Find(".tags a").Each(func(i int, s *goquery.Selection) {
		if href, b := s.Attr("href"); b {
			tag := Tag{}
			tag.Url = href
			tag.Name = s.Text()
			tags = append(tags, tag)
		}
	})
	//doc.Find(".nava").Each(func(i int, s *goquery.Selection) {
	//	find := s.Find("a")
	//	if href, b := find.Attr("href"); b {
	//		tag := Tag{}
	//		tag.Url = baseUrl + href
	//		tag.Title = find.Text()
	//		tags = append(tags, tag)
	//	}
	//})
	return
}

func (t Tag) Choose() string {
	return t.Name
}

func (a Album) Choose() string {
	return a.Title
}

func (t *Tag) PageUrl(page int) string {
	return fmt.Sprintf("%s&page=%d", t.Url, page)
}

func (a Album) LocalDir() (dir string, err error) {
	dir = path.Join(imagesBaseDir, fmt.Sprintf("%s(%d)", a.Title, a.Count))
	if _, err = os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return
		}
	}
	return
}
