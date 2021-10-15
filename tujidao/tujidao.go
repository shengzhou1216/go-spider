package tujidao

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"go-spider/common"
	"go-spider/downloader"
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
	Hint                    = "选择标签(T/t)选择页码(P/p),下载(D/d{page})"
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
	SourceTag    Tag
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
	fmt.Println("可选标签如下: ")
	for ix, tag := range tags {
		if ix%10 == 0 {
			fmt.Println()
		}
		fmt.Print(fmt.Sprintf("(%d)%s |", ix+1, tag.Name))
	}
	tagIndex := 0
	fmt.Println("请选择一个tag(序号): ")
	for {
		_, err := fmt.Scanln(&tagIndex)
		if err != nil {
			fmt.Printf("无效输入:%v，请重新输入", err)
			continue
		}
		if tagIndex < 1 || tagIndex > len(tags) {
			fmt.Println("超出范围了，请重新选择标签。")
			continue
		}
		break
	}
	return tags[tagIndex-1]
}

func choosePage(pages int) int {
	var page int
	for {
		fmt.Println("请输入要查看的页码：")
		if _, err := fmt.Scanln(&page); err != nil {
			fmt.Println("无效页码！请重新输入")
			continue
		}
		if page < 1 || page > pages {
			fmt.Printf("请输入1-%d范围内的数字", pages)
			continue
		}
		break
	}
	fmt.Printf("第%d页中的相册如下: ", page)
	return page
}

func hint() {
	fmt.Println()
	fmt.Println(Hint)
}

func TujidaoSpider() {
	client := &http.Client{}
	tags, _ := getTagsAndCategories(client)
	for {
	ChooseTag:
		// 选择tag
		tag := chooseTag(tags)
		fmt.Printf("你选择的标签是:%s", tag.Name)
		// 获取tag下的页数
		pages := tag.getPages(client)
		if pages == 0 {
			fmt.Println("此标签中没有数据，请重新选择标签")
			continue
		}
	ChoosePage:
		// 选择page
		fmt.Printf("标签【%s】下共有%d页\n", tag.Name, pages)
		page := choosePage(pages)
	AfterChoosePage:
		fmt.Printf("第%d页中的相册如下: ", page)
		// 列出相册
		albums := tag.listAlbums(client, tag.PageUrl(page))
		for ix, a := range albums {
			fmt.Printf("(%d)%s(%d)\n", ix+1, a.Title, a.Count)
		}
	ChoosePageOrDownload:
		fmt.Println("下载此页请输入D.要下载指定页请输入DP，其中P为页码；如D1，表示下载第1页。输入数字可切换页码，D1-2,表示1-2页；DA/Da: 表示所有页。T/t:回到tag选择")
		var cmd string
		var downloadPages []int
		var startPage int
		var endPage int
		var isDownload bool
		if _, err := fmt.Scanln(&cmd); err != nil {
			fmt.Println(err)
			continue
		}
		if strings.ToLower(cmd) == "t" {
			goto ChooseTag
		}
		re := regexp.MustCompile(`[D|d](\S*)`)
		if re.Match([]byte(cmd)) {
			// 下载
			matchs := re.FindSubmatch([]byte(cmd))
			sp := matchs[1]
			if len(sp) == 0 {
				fmt.Printf("您选择了下载当前页：%d。", page)
				// 下载当前页
				startPage = page
				endPage = page
			} else {
				// 下载指定页
				if strings.ToLower(string(sp)) == "a" {
					//下载所有页
					fmt.Printf("您选择了下载所有页，此标签下共有%d页。", tag.Pages)
					startPage = 1
					endPage = tag.Pages
				} else {
					re = regexp.MustCompile(`\d*-\d*`)
					if re.Match(sp) {
						// 下载指定范围的页
						sps := string(sp)
						startPages := strings.Split(sps, "-")[0]
						endPages := strings.Split(sps, "-")[1]
						if len(startPages) == 0 {
							startPage = 1
						} else {
							if p, err := strconv.ParseInt(startPages, 10, 0); err != nil {
								fmt.Println(err)
								goto ChoosePageOrDownload
							} else {
								if p < 1 || int(p) > pages {
									fmt.Println("无效起始页码，请重新输入")
									goto ChoosePageOrDownload
								} else {
									// download
									startPage = int(p)
								}
							}
						}
						if len(endPages) == 0 {
							endPage = tag.Pages
						} else {
							if p, err := strconv.ParseInt(endPages, 10, 0); err != nil {
								fmt.Println(err)
								goto ChoosePageOrDownload
							} else {
								if p < 1 || int(p) > pages {
									fmt.Println("无效结束页码，请重新输入")
									goto ChoosePageOrDownload
								} else {
									// download
									endPage = int(p)
								}
							}
						}
						fmt.Printf("您选择了下载%d-%d页。", startPage, endPage)
					} else {
						// 下载指定页
						if p, err := strconv.ParseInt(string(sp), 10, 0); err != nil {
							fmt.Println(err)
							goto ChoosePageOrDownload
						} else {
							if p < 1 || int(p) > pages {
								fmt.Println("无效页码，请重新输入")
								goto ChoosePageOrDownload
							} else {
								// download
								startPage = int(p)
								endPage = int(p)
							}
						}
						fmt.Printf("您选择了下载第%d页。", startPage)
					}
				}
			}
			for ; startPage <= endPage; startPage++ {
				downloadPages = append(downloadPages, startPage)
			}
			if len(downloadPages) > 0 {
				isDownload = true
			} else {
				fmt.Printf("没有要下载的页面.起始页：%d,结束页：%d", startPage, endPage)
			}
		} else {
			// 页码
			if p, err := strconv.ParseInt(cmd, 10, 0); err != nil {
				fmt.Println(err)
				goto ChoosePageOrDownload
			} else {
				page = int(p)
				goto AfterChoosePage
			}
		}
		if isDownload {
			// 初始化下载器
			downloader := downloader.NewDownloader()
			downloadAlbums := tag.listAlbums(client, tag.PagesUrl(downloadPages)...)
			// 添加任务
			for _, a := range downloadAlbums {
				if err := AddAlbumTask(&downloader, &a); err != nil {
					fmt.Println(err)
					continue
				}
			}
			// 下载相册
			downloader.Start()
			downloader.Result()
			// 回到选择page
			goto ChoosePage
		}
	}
}

// AddAlbumTask 将相册添加到任务中
func AddAlbumTask(downloader *downloader.Downloader, album *Album) (err error) {
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
func (t *Tag) listAlbums(client *http.Client, urls ...string) (albums []Album) {
	headers := map[string]string{"cookie": cookie}
	for _, url := range urls {
		doc := requestDocument(client, url, headers)
		doc.Find(".hezi ul li").Each(func(i int, li *goquery.Selection) {
			if id, exists := li.Attr("id"); exists {
				idd, err := strconv.ParseInt(id, 10, 0)
				if err != nil {
					fmt.Println(err)
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
					Id:        int(idd),
					Count:     int(count),
					SourceTag: *t,
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
	}
	return
}

// 请求html，返回document对象
func requestDocument(client *http.Client, url string, headers map[string]string) (doc *goquery.Document) {
	if !strings.HasPrefix(url, "http") {
		url = baseUrl + url
	}
	req, err := common.FormRequest(url, headers)
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

func (t Tag) PagesUrl(pages []int) (r []string) {
	for _, p := range pages {
		r = append(r, t.PageUrl(p))
	}
	return
}

func (a Album) LocalDir() (dir string, err error) {
	dir = path.Join(imagesBaseDir, a.SourceTag.Name, fmt.Sprintf("%s(%d)", a.Title, a.Count))
	if _, err = os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return
		}
	}
	return
}
