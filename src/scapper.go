package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

func scrapDocs(docsetDocumentsPath string, db *sql.DB) {
	c := colly.NewCollector(
		colly.AllowedDomains("docs.unrealengine.com"),
		colly.Async(true),
		colly.DisallowedURLFilters(regexp.MustCompile(".*((\\.zip)|(\\.rar)|(\\.pdf)|(\\.max))$"), regexp.MustCompile("https:\\/\\/docs\\.unrealengine\\.com\\/en-US\\/((API)|(BlueprintAPI)|(PythonAPI))\\/.*\\/")),
	)
	c.SetRequestTimeout(20 * time.Second)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*docs.unrealengine.*",
		Parallelism: 4,
		Delay:       55 * time.Millisecond,
	})

	c.OnRequest(func(r *colly.Request) {
		//fmt.Println("Visiting", r.URL)
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Println("Something went wrong visting", r.Request.URL, err)
	})

	c.OnResponse(func(r *colly.Response) {
		// fmt.Println("Visited", r.Request.URL)
	})

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		e.Request.Visit(e.Attr("href"))
	})

	c.OnHTML("img[src]", func(e *colly.HTMLElement) {
		e.Request.Visit(e.Attr("src"))
	})

	c.OnScraped(func(r *colly.Response) {
		relativeLocalPath := strings.Replace(r.Request.URL.String(), r.Request.URL.Hostname(), "", 1)
		relativeLocalPath = strings.Replace(relativeLocalPath, r.Request.URL.Scheme+"://", "", 1)

		absoluteLocalPath := filepath.Join(docsetDocumentsPath, relativeLocalPath)

		err := os.MkdirAll(filepath.Dir(absoluteLocalPath), 0755)
		if err != nil {
			fmt.Println(err)
		}

		if filepath.Ext(r.Request.URL.String()) == ".html" {
			transformedHTML, entryName, entryType, err := transformHTML(r.Request.URL.String(), bytes.NewReader(r.Body))
			if err != nil {
				fmt.Println(err)
			}

			r.Body = []byte(transformedHTML)

			entryPath, _ := filepath.Rel(docsetDocumentsPath, absoluteLocalPath)
			addEntryToDatabase(db, entryName, entryType, entryPath)
		}

		err = r.Save(absoluteLocalPath)
		if err != nil {
			fmt.Println(err)
		}
	})

	c.Visit("https://docs.unrealengine.com/en-US/SiteIndex/index.html")

	c.Wait()
}
