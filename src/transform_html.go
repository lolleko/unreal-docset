package main

import (
	"io"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
)

type entry struct {
	html           string
	name           string
	ttype          string
	isValid        bool
	ommitFromIndex bool
}

func transformHTML(htmlPath string, r io.Reader) (entry, error) {

	minifier := minify.New()
	minifier.AddFunc("text/css", css.Minify)
	minifier.AddFunc("text/html", html.Minify)

	doc, err := goquery.NewDocumentFromReader(r)

	if err != nil {
		log.Fatal(err)
	}

	// remove some bloat
	doc.Find(".crumbs").Each(func(i1 int, s1 *goquery.Selection) {
		doc.Find("#osContainer").Each(func(i2 int, s2 *goquery.Selection) {
			s2.AfterNodes(s1.Clone().Nodes...)
		})
	})

	doc.Find("#page_head, #navWrapper, #splitter, #footer, #skinContainer, #feedbackButton, #feedbackMessage, #osContainer, #announcement, #courses").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	// change abolsute urls to docs.unrealengine.com to relative urls
	doc.Find("a[href*='docs.unrealengine.com']").Each(func(i int, s *goquery.Selection) {
		s.SetAttr("href", resolveAbsoluteRef(s.AttrOr("href", ""), htmlPath))
	})

	// Newer UE docs use lazy loading we do not need that on a static site
	doc.Find("img[data-src]").Each(func(i int, s *goquery.Selection) {
		s.SetAttr("src", s.AttrOr("data-src", ""))
		s.RemoveAttr("data-src")
		s.RemoveClass("lazy-load")
	})

	// add some exceptions for dark mode
	doc.Find(".hero, .graph, iframe.embedded_video, .topics.item .subject, .topics.item .img_container, .topics.item img, .topics.item .color_container, .hero.background .imgContainer img").Each(func(i int, s *goquery.Selection) {
		s.AddClass("dash-ignore-dark-mode")
	})

	// Parse toc

	// doc.Find("#toc_link").Each(func(i int, s *goquery.Selection) {
	// 	s.SetAttr("name", "//apple_ref/cpp/Section/"+url.QueryEscape(s.Text()))
	// 	s.AddClass("dashAnchor")
	// })

	// inject styles
	doc.Find("head").Each(func(i int, s *goquery.Selection) {
		for _, cssFile := range extraCSSFiles {
			s.AppendHtml(`<link rel="stylesheet" href=" ` + resolveAbsoluteRef(cssFile, htmlPath) + ` ">`)
		}
		s.AppendHtml(`<link rel="stylesheet" href=" ` + resolveAbsoluteRef("/Include/CSS/dash_style_overrides.css", htmlPath) + ` ">`)
	})

	// remove js
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	html, err := doc.Html()
	if err != nil {
		panic(err)
	}
	html, err = minifier.String("text/html", html)
	if err != nil {
		panic(err)
	}

	entryName, entryType, isValid, ommitFromIndex := extractNameAndType(doc, htmlPath)

	return entry{
		html:           html,
		name:           entryName,
		ttype:          entryType,
		isValid:        isValid,
		ommitFromIndex: ommitFromIndex,
	}, err
}

func extractNameAndType(doc *goquery.Document, htmlPath string) (string, string, bool, bool) {
	// determine name and type

	entryType := ""

	entryName := filepath.Base(htmlPath)

	isValid := true

	ommitFromIndex := false

	if strings.Contains(strings.ToUpper(htmlPath), strings.ToUpper("/BlueprintAPI/")) {
		doc.Find("#actions").Each(func(i int, s *goquery.Selection) {
			entryType = "Category"
		})

		// if entryType == "" {
		// 	doc.Find(".graph").Each(func(i int, s *goquery.Selection) {
		// 		entryType = "Node"
		// 	})
		// }

		// default type node
		if entryType == "" {
			entryType = "Node"
		}

		doc.Find("meta[name='title']").Each(func(i int, s *goquery.Selection) {
			entryName, _ = s.Attr("content")
		})
	} else {
		doc.Find(".simplecode_api").Each(func(i int, s *goquery.Selection) {
			syntaxText := strings.TrimSpace(s.Text())

			matchedClass, err := regexp.MatchString(`(?m)^class\s+[UAS]\w*`, syntaxText)

			if err == nil && matchedClass {
				entryType = "Class"
				return
			}

			matchedStruct, err := regexp.MatchString(`(?m)(^|>)((struct)|(class))\s+(F|T)\w*`, syntaxText)

			if err == nil && matchedStruct {
				entryType = "Struct"
				return
			}

			matchedInterface, err := regexp.MatchString(`(?m)^class\s+I\w*`, syntaxText)

			if err == nil && matchedInterface {
				entryType = "Interface"
				return
			}

			matchedEnum, err := regexp.MatchString(`(?m)^enum\s+(class\s+)?E\w*\s*{`, syntaxText)

			if err == nil && matchedEnum {
				entryType = "Enum"
				return
			}

			matchedProperty, err := regexp.MatchString(`(?m)^UPROPERTY\(`, syntaxText)

			if err == nil && matchedProperty {
				entryType = "Property"
				return
			}

			entryType = "Field"
		})

		if entryType == "" {
			doc.Find(".heading.expanded").Each(func(i int, s *goquery.Selection) {
				if strings.Contains(s.Text(), "Filters") {
					entryType = "Category"
					return
				}
				if strings.Contains(s.Text(), "Classes") {
					entryType = "Module"
					return
				}
			})
		}

		// API pages should be category by default
		if entryType == "" && strings.Contains(strings.ToUpper(htmlPath), strings.ToUpper("/API/")) {
			entryType = "Category"
		}

		// everything else is a guide
		if entryType == "" {
			entryType = "Guide"
		}

		// Normal method
		doc.Find("meta[name='title']").Each(func(i int, s *goquery.Selection) {
			entryName, _ = s.Attr("content")
			if strings.Contains(entryName, "::") {
				entryType = "Method"
			}

			// Overloaded method
			parts := strings.Split(htmlPath, "/")
			if len(parts) >= 2 && unicode.IsDigit([]rune(parts[len(parts)-2])[0]) {
				ommitFromIndex = true
			}
		})

		// Overload list
		doc.Find(".info").Each(func(i int, s *goquery.Selection) {
			if strings.Contains(s.Text(), "Overload list") {
				entryName = filepath.Base(filepath.Dir(filepath.Dir(htmlPath))) + "::" + entryName
				entryType = "Method"
			}
		})
	}

	return entryName, entryType, isValid, ommitFromIndex
}

func resolveAbsoluteRef(absoluteRef string, htmlPath string) string {
	strippedRef := strings.ReplaceAll(absoluteRef, "https://docs.unrealengine.com", "")
	// fix some broken links
	strippedRef = strings.ReplaceAll(strippedRef, "https:///docs.unrealengine.com", "")
	strippedRef = strings.ReplaceAll(strippedRef, "http://docs.unrealengine.com", "")
	strippedRef = strings.ReplaceAll(strippedRef, "https://", "")
	strippedRef = strings.ReplaceAll(strippedRef, "/latest/INT/", "/en-US/")

	strippedURL := strings.ReplaceAll(htmlPath, "https://docs.unrealengine.com", "")
	// TODO docset name should not be hardcoded
	strippedURL = strings.ReplaceAll(strippedURL, "UnrealEngine4.docset/Contents/Resources/Documents", "")
	strippedURL = filepath.Dir(strippedURL)

	relRef, err := filepath.Rel(strippedURL, strippedRef)
	if err != nil {
		log.Println(err)
	}

	return relRef
}
