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

type Chooser interface {
	Choose() string
}

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

func TujidaoSpiderRun() {
	client := &http.Client{}
	tags, _ := getTagsAndCategories(client)
	fmt.Println("共获取到如下tag:")
	for ix, tag := range tags {
		if ix > 0 && ix%10 == 0 {
			fmt.Println()
		}
		fmt.Printf("(%d)%s | ", ix+1, tag.Name)
	}
	tagIndex := 0
	fmt.Println("\n请选择一个tag(序号): ")
	_, err := fmt.Scanln(&tagIndex)
	if err != nil {
		log.Fatal(err)
	}
	// 获取tag下的页数
	tag := tags[tagIndex-1]
	fmt.Println("你选择的标签是:", tag.Name)
	pages := tag.getPages(client)
	if pages == 0 {
		fmt.Println("此标签中没有数据。")
		return
	}
	fmt.Printf("此标签下共有%d页\n", pages)
	fmt.Println("第一页相册: ")
	// 获取第一页的相册
	albums := tag.listAlbums(client, tag.Url)
	for ix, a := range albums {
		fmt.Printf("(%d)%s(%d) \n", ix+1, a.Title, a.Count)
	}
	fmt.Println("请输入要下载的页码： ")
	downloadPage := 0
	if _, err := fmt.Scanln(&downloadPage); err != nil {
		log.Fatal(err)
	}
	// 初始化下载器
	downloader := NewDownloader()
	downloadAlbums := tag.listAlbums(client, tag.PageUrl(downloadPage))
	// 添加任务
	for _, a := range downloadAlbums {
		if err := AddAlbumTask(&downloader, &a); err != nil {
			log.Fatal(err)
		}
	}
	// 下载相册
	downloader.Start()
	downloader.Result()
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
	//fmt.Println(doc.Html())
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
	dir = path.Join(imagesBaseDir, fmt.Sprintf("%s", a.Title))
	if _, err = os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0666); err != nil {
			return
		}
	}
	return
}
