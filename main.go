package main

import (
	"fmt"
	"goquery/mysql"
	"log"
	"net/http"
	"regexp"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

type News struct {
	Title   string
	Media   string
	Url     string
	PubTime string
	Content string
}

func main() {
	url := "https://www.gamersky.com/news/"
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	html, err := goquery.NewDocumentFromReader(resp.Body)
	var newsList []string
	newsList = getNewsList(html, newsList)

	var wg sync.WaitGroup
	log.Println( len(newsList))
	for i := 0; i < len(newsList); i++ {
		wg.Add(1)
		go getNews(newsList[i], &wg)
	}
	wg.Wait()
}

func getNewsList(html *goquery.Document, newsList []string) []string {
	// '//a[@class="tt"]/@href'
	html.Find("a[class=tt]").Each(func(i int, selection *goquery.Selection) {
		url, _ := selection.Attr("href")
		newsList = append(newsList, url)
	})
	return newsList
}

func getNews(url string, wg *sync.WaitGroup) {
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		wg.Done()
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error: status code %d", resp.StatusCode)
		wg.Done()
		return
	}

	html, err := goquery.NewDocumentFromReader(resp.Body)
	news := News{}

	news.Url = url
	news.Media = "GameSky"
	html.Find("div[class=Mid2L_tit]>h1").Each(func(i int, selection *goquery.Selection) {
		news.Title = selection.Text()
	})

	if news.Title == "" {
		wg.Done()
		return
	}

	html.Find("div[class=Mid2L_con]>p").Each(func(i int, selection *goquery.Selection) {
		news.Content = news.Content + selection.Text()
	})

	var tmpTime string
	html.Find("div[class=detail]").Each(func(i int, selection *goquery.Selection) {
		tmpTime = selection.Text()
	})
	reg := regexp.MustCompile(`\d+`)
	timeString := reg.FindAllString(tmpTime, -1)
	news.PubTime = fmt.Sprintf("%s-%s-%s %s:%s:%s", timeString[0], timeString[1], timeString[2], timeString[3], timeString[4], timeString[5])

	db := mysql.DBCon()

	stmt, err := db.Prepare(
		"insert into gamesky (`title`, `url`, `media`, `content`, `pub_time`) values (?,?,?,?,?)")
	if err != nil {
		log.Println(err)
		wg.Done()
	}
	defer stmt.Close()

	rs, err := stmt.Exec(news.Title, news.Url, news.Media, news.Content, news.PubTime)
	if err != nil {
		log.Println(err)
		wg.Done()
	}
	if id, _ := rs.LastInsertId(); id > 0 {
		log.Println("插入成功")
	}
	wg.Done()
}
