package main

import (
	"fmt"
	hhtml "html"
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

	doc.Find("script, #page_head, #navWrapper, #splitter, #footer, #skinContainer, #feedbackButton, #feedbackMessage, #osContainer, #announcement, #courses, meta[name*='course'], meta[name*='twitter:card']").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	// change abolsute urls to docs.unrealengine.com, to relative urls
	doc.Find("a[href*='docs.unrealengine.com']").Each(func(i int, s *goquery.Selection) {
		s.SetAttr("href", resolveAbsoluteRef(s.AttrOr("href", ""), htmlPath))
	})

	// Newer UE docs use lazy loading we do not need that on a static site
	doc.Find("img[data-src]").Each(func(i int, s *goquery.Selection) {
		s.SetAttr("src", s.AttrOr("data-src", ""))
		s.RemoveAttr("data-src")
		s.RemoveClass("lazyload")
	})

	// Remove picture tags, those cause issues in dash darkmode
	doc.Find("picture").Each(func(i int, s *goquery.Selection) {
		childrenHTML, _ := s.Html()
		s.ReplaceWithHtml(fmt.Sprintf("<div>%s</div>", childrenHTML))
	})

	// add some exceptions for dark mode
	doc.Find(".topics.item .subject, .graph").Each(func(i int, s *goquery.Selection) {
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

	// remove markdown links
	doc.Find("body").Each(func(i int, s *goquery.Selection) {
		bodyHTML, err := s.Html()

		if err != nil {
			panic(err)
		}

		// TODO move regex so we dont compile everytime
		linkRegexp := regexp.MustCompile(`\[(.*?)\]\((.*?)\)`)

		htmlWithoutMarkdown := linkRegexp.ReplaceAllStringFunc(bodyHTML, func(match string) string {
			submatches := linkRegexp.FindAllStringSubmatch(match, -1)
			// we dont have todo any nil checking here ebcause we already now how the FindAllStringSubmatch will lokk like
			// this is done to modify the link before insertion (cant be done with $2/$1 replacement directly)

			// TODO dont hardcode base path, but tbh im not sure which onvention those md links use sicne they are broken in the online version aswell
			// just guessing en-US for now
			return fmt.Sprintf(`<a href="%s">%s</a>`, resolveAbsoluteRef("/en-US/"+strings.ReplaceAll(submatches[0][2], "\\", "/")+"/index.html", htmlPath), submatches[0][1])
		})
		s.SetHtml((htmlWithoutMarkdown))
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
			trimmedContents := strings.TrimSpace(s.Text())

			// TODO refactor this into a lookup table (a lot of replicated code here)
			matchedClass, err := regexp.MatchString(`(?m)(^|\))class\s+(PAPER2)?[UAS]\w*`, trimmedContents)

			if err == nil && matchedClass {
				entryType = "Class"
				return
			}

			matchedStruct, err := regexp.MatchString(`(?m)(^|>)((struct)|(class))\s+(F|T)\w*`, trimmedContents)

			if err == nil && matchedStruct {
				entryType = "Struct"
				return
			}

			matchedInterface, err := regexp.MatchString(`(?m)(^|\))class\s+I\w*`, trimmedContents)

			if err == nil && matchedInterface {
				entryType = "Interface"
				return
			}

			matchedEnum, err := regexp.MatchString(`(?m)(^|\))enum\s+(class\s+)?E\w*\s*{`, trimmedContents)

			if err == nil && matchedEnum {
				entryType = "Enum"
				return
			}

			matchedProperty, err := regexp.MatchString(`(?m)^UPROPERTY\(`, trimmedContents)

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
			entryName = hhtml.UnescapeString(entryName)
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

		// Ommit fields from index, they just clutter the index and are not really useful
		// In additional there are a lot of fields with duplicate names
		if entryType == "Field" {
			ommitFromIndex = true
		}
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
