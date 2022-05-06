package main

import (
	"errors"
	"fmt"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

type Course struct {
	Name            string
	Url             string
	Collector       *colly.Collector
	LectureNoteUrls []string
	Categories      []string
}

func (c Course) Mkdir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (c Course) dir() string {
	dir := fmt.Sprintf("courses/%s", c.Name)
	c.Mkdir(dir)
	return dir
}

func (c Course) lectureNotesDir() string {
	dir := fmt.Sprintf("%s/lecture-notes", c.dir())
	c.Mkdir(dir)
	return dir
}
func (c Course) quizzesDir() string {
	dir := fmt.Sprintf("%s/quizzes", c.dir())
	c.Mkdir(dir)
	return dir
}

func (c Course) problemsDir() string {
	dir := fmt.Sprintf("%s/problems", c.dir())
	c.Mkdir(dir)
	return dir
}
func (c Course) assignmentsDir() string {
	dir := fmt.Sprintf("%s/assignments", c.dir())
	c.Mkdir(dir)
	return dir
}

func (c Course) isInCategory(str string) bool {
	str = strings.ToLower(strings.TrimSpace(str))
	for _, cat := range c.Categories {
		cat = strings.ToLower(strings.TrimSpace(cat))
		if str == cat || strings.Index(cat, str) >= 0 {
			return true
		}
	}
	return false
}

var isLectureNotes = is("Lecture Notes")
var isProblems = is("Practice Problems")
var isAssignments = is("Assignments")
var isQuizzes = is("Quizzes")

func is(cat string) func(string) bool {
	return func(str string) bool {
		str = strings.ToLower(strings.TrimSpace(str))
		cat = strings.ToLower(strings.TrimSpace(cat))
		return str == cat || strings.Index(cat, str) > 0
	}
}

func NewCourse(name, url string, collector *colly.Collector) Course {
	return Course{
		Name:      name,
		Url:       url,
		Collector: collector,
		Categories: []string{
			"Lecture Notes",
			"Quizzes",
			"Practice Problems",
			"Assignments",
		},
	}
}

func (c Course) find(collector *colly.Collector) {
	collector.OnXML("/html[1]/body[1]/div[1]/div[6]/div[1]/div[1]/div[1]/nav[1]/ul[1]/li/div/span/a", func(element *colly.XMLElement) {
		text := strings.TrimSpace(element.Text)
		href := element.Attr("href")
		href = element.Request.AbsoluteURL(href)
		if c.isInCategory(text) {
			collector2 := collector.Clone()
			log.Printf("text:%s,href:%s \n", text, href)
			if isLectureNotes(text) {
				c.findLectureNotes(collector2, href)
			} else if isProblems(text) {
				c.findProblems(collector2, href)
			} else if isQuizzes(text) {
				c.findQuizzes(collector2, href)
			} else if isAssignments(text) {
				c.findAssignments(collector2, href)
			}
		}
	})
	collector.OnRequest(func(request *colly.Request) {
		log.Printf("Visiting: %s\n", c.Url)
	})
	// Set error handler
	collector.OnError(func(r *colly.Response, err error) {
		fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})
	collector.Visit(c.Url)
}

func (c Course) findSource(collector *colly.Collector, xpath, saveDir, url string) {
	var wg sync.WaitGroup
	collector.OnXML(xpath, func(element *colly.XMLElement) {
		text := strings.TrimSpace(element.Text)
		href := element.Attr("href")
		href = element.Request.AbsoluteURL(href)
		log.Printf("text:%s,href:%s \n", text, href)
		if href != "" {
			wg.Add(1)
			go func() {
				defer wg.Done()
				c.downloadFile(collector.Clone(), saveDir, text, href)
			}()
		}
	})
	collector.OnRequest(func(request *colly.Request) {
		log.Printf("Visiting: %s\n", url)
	})
	// Set error handler
	collector.OnError(func(r *colly.Response, err error) {
		fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})
	collector.Visit(url)
	wg.Wait()
}

// find lecture notes url
func (c Course) findLectureNotes(collector *colly.Collector, url string) {
	c.findSource(collector, "//main[@id='course-content-section']/table/tbody/tr/td/a", c.lectureNotesDir(), url)
}

// find quizzes url
func (c Course) findQuizzes(collector *colly.Collector, url string) {
	c.findSource(collector, "/html[1]/body[1]/div[1]/div[7]/div[1]/div[3]/main[1]/div[1]/div[1]/div[1]/div[1]/article[1]/main[1]/table[1]/tbody[1]/tr/td/p/a", c.quizzesDir(), url)
}

// find quizzes url
func (c Course) findProblems(collector *colly.Collector, url string) {
	c.findSource(collector, "/html[1]/body[1]/div[1]/div[7]/div[1]/div[3]/main[1]/div[1]/div[1]/div[1]/div[1]/article[1]/main[1]/table[1]/tbody[1]/tr/td/a", c.problemsDir(), url)
}

// find quizzes url
func (c Course) findAssignments(collector *colly.Collector, url string) {
	c.findSource(collector, "/html[1]/body[1]/div[1]/div[7]/div[1]/div[3]/main[1]/div[1]/div[1]/div[1]/div[1]/article[1]/main[1]/table[1]/tbody[1]/tr/td/p/a", c.assignmentsDir(), url)
}

// get one lecture note
func (c Course) downloadFile(collector *colly.Collector, dir, title, url string) {
	collector.OnXML("//a[@class='download-file']", func(element *colly.XMLElement) {
		fileUrl := element.Attr("href")
		ext := path.Ext(fileUrl)
		name := fmt.Sprintf("%s%s", title, ext)
		err := download(path.Join(dir, name), element.Request.AbsoluteURL(fileUrl))
		if err != nil {
			log.Printf("download lecture note err:%s \n", err)
		}
	})
	collector.OnRequest(func(request *colly.Request) {
		log.Printf("Visiting: %s\n", url)
	})
	// Set error handler
	collector.OnError(func(r *colly.Response, err error) {
		fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})
	collector.Visit(url)
}

// reset invalid filename to valid
func (c Course) resetFileName() {
	filepath.WalkDir(c.dir(), func(p string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			name := d.Name()
			oldName := p
			base := path.Dir(p)
			name = strings.ReplaceAll(name, ":", "-")
			name = strings.ReplaceAll(name, "?", "-")
			name = strings.ReplaceAll(name, ",", "-")
			name = strings.ReplaceAll(name, "/", "-")
			name = strings.ReplaceAll(name, "\\", "-")
			name = strings.ReplaceAll(name, "<", "-")
			name = strings.ReplaceAll(name, ">", "-")
			name = strings.ReplaceAll(name, "|", "-")
			name = strings.ReplaceAll(name, "\"", "-")
			name = strings.ReplaceAll(name, "*", "-")
			newName := path.Join(base, name)
			err := os.Rename(oldName, newName)
			if err != nil {
				log.Println(err)
			}
		}
		return err
	})
}

func download(name, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
		return errors.New(resp.Status)
	}
	file, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	c := colly.NewCollector(colly.Debugger(&debug.LogDebugger{}))
	c.WithTransport(&http.Transport{
		Proxy: http.ProxyURL(&url.URL{
			Host: "localhost:7890",
		}),
	})
	c.UserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36"
	name := "Introduction To Algorithm"
	homeUrl := "https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/"
	course := NewCourse(name, homeUrl, c)
	course.find(c)
	course.resetFileName()
}
