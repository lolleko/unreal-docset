package main

import (
	"fmt"
	"log"

	"github.com/gocolly/colly"
)

func main() {
	collector := colly.NewCollector(
		colly.AllowedDomains("docs.unrealengine.com"),
	)

	collector.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})

	collector.OnError(func(_ *colly.Response, err error) {
		log.Println("Something went wrong:", err)
	})

	collector.OnResponse(func(r *colly.Response) {
		fmt.Println("Visited", r.Request.URL)
	})

	collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		e.Request.Visit(e.Attr("href"))
	})

	collector.OnScraped(func(r *colly.Response) {
		fmt.Println("Finished", r.Request.URL)
	})

	collector.Visit("https://docs.unrealengine.com/en-US/SiteIndex/index.html")
}
