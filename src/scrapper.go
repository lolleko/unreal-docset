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

var extraCSSFiles []string

func scrapDocs(docsetDocumentsPath string, db *sql.DB) {
	extraCSSFiles = []string{
		"https://docs.unrealengine.com/Include/CSS/jquery-ui.min.css",
		"https://docs.unrealengine.com/Include/CSS/jquery.fancybox.css",
		"https://docs.unrealengine.com/Include/CSS/jquery.qtip.css",
		"https://docs.unrealengine.com/Include/CSS/jquery.recommendations.css",
		"https://docs.unrealengine.com/Include/CSS/skin_overrides.css",
		"https://docs.unrealengine.com/Include/CSS/twentytwenty.css",
		"https://docs.unrealengine.com/Include/CSS/udn_public.css",
	}

	// retried := new(sync.Map)

	c := colly.NewCollector(
		colly.AllowedDomains("docs.unrealengine.com"),
		colly.Async(true),
		// "https:\\/\\/docs\\.unrealengine\\.com\\/en-US\\/((API)|(BlueprintAPI)|(PythonAPI))\\/.*\\/"
		colly.DisallowedURLFilters(regexp.MustCompile(".*((\\.zip)|(\\.rar)|(\\.pdf)|(\\.max))$"), regexp.MustCompile("https:\\/\\/docs\\.unrealengine\\.com\\/en-US\\/((PythonAPI))\\/.*\\/")),
	)
	c.SetRequestTimeout(20 * time.Second)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*docs.unrealengine.*",
		Parallelism: 16,
		Delay:       5 * time.Millisecond,
	})

	c.OnRequest(func(r *colly.Request) {
		//fmt.Println("Visiting", r.URL)
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Println("Something went wrong visting", r.Request.URL, err)

		// _, hasRetried := retried.Load(r.Request.URL.String())
		// if !hasRetried {
		// 	retried.Store(r.Request.URL.String(), true)
		// 	r.Request.Retry()
		// } else {
		// 	log.Println("Already retried", r.Request.URL.String())
		// }
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

	c.OnHTML("link[href][rel='icon']", func(e *colly.HTMLElement) {
		e.Request.Visit(e.Attr("href"))
	})

	c.OnHTML("img[data-src]", func(e *colly.HTMLElement) {
		e.Request.Visit(e.Attr("data-src"))
	})

	c.OnScraped(func(r *colly.Response) {
		relativeLocalPath := strings.Replace(r.Request.URL.String(), r.Request.URL.Hostname(), "", 1)
		relativeLocalPath = strings.Replace(relativeLocalPath, r.Request.URL.Scheme+"://", "", 1)

		absoluteLocalPath := filepath.Join(docsetDocumentsPath, relativeLocalPath)

		if !strings.ContainsAny(absoluteLocalPath, "?") && !strings.ContainsAny(absoluteLocalPath, "=") {
			err := os.MkdirAll(filepath.Dir(absoluteLocalPath), 0755)
			if err != nil {
				fmt.Println(err)
			}

			if filepath.Ext(r.Request.URL.String()) == ".css" {
				re := regexp.MustCompile(`url\(['"](.+?)['"]\)`)
				matches := re.FindAllStringSubmatch(string(r.Body), -1)
				for _, match := range matches {
					fmt.Println("Visisting CSS url ", match[1])
					r.Request.Visit(match[1])
				}
			}

			if filepath.Ext(r.Request.URL.String()) == ".html" {
				entry, err := transformHTML(r.Request.URL.String(), bytes.NewReader(r.Body))
				if err != nil {
					fmt.Println(err)
				}
				if entry.isValid {
					r.Body = []byte(entry.html)

					entryPath, _ := filepath.Rel(docsetDocumentsPath, absoluteLocalPath)

					if !entry.ommitFromIndex {
						addEntryToDatabase(db, entry.name, entry.ttype, entryPath)
					}
				}
			}

			err = r.Save(absoluteLocalPath)
			if err != nil {
				fmt.Println(err)
			}
		}
	})

	for _, cssFile := range extraCSSFiles {
		c.Visit(cssFile)
	}

	c.Visit("https://docs.unrealengine.com/en-US/SiteIndex/index.html")

	c.Wait()
}
