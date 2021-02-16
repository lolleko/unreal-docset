package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {

	var outDir string
	flag.StringVar(&outDir, "outDir", ".", "OutputDirectory")

	var resourceDir string
	flag.StringVar(&resourceDir, "resourceDir", "./resources", "ResourceDirectory")

	flag.Parse()

	_, err := os.Stat(resourceDir)
	if os.IsNotExist(err) {
		panic("Resource dir not found please specify with --resourceDir [PathToResourceDir]")
	}

	const docsetName = "UnrealEngine4.docset"

	docsetPath := filepath.Join(outDir, docsetName)

	docsetContentsPath := filepath.Join(docsetPath, "Contents/")

	docsetResourcesPath := filepath.Join(docsetContentsPath, "Resources/")

	docsetDocumentsPath := filepath.Join(docsetResourcesPath, "Documents/")

	db, err := initDatabase(filepath.Join(docsetResourcesPath, "docSet.dsidx"))
	if err != nil {
		fmt.Println(err)
	}

	err = os.MkdirAll(filepath.Dir(docsetDocumentsPath), 0755)
	if err != nil {
		fmt.Println(err)
	}

	// Copy plist and css

	copyFile(filepath.Join(resourceDir, "Info.plist"), filepath.Join(docsetContentsPath, "Info.plist"))

	os.MkdirAll(filepath.Join(docsetDocumentsPath, "Include/CSS/"), 0755)
	copyFile(filepath.Join(resourceDir, "dash_style_overrides.css"), filepath.Join(docsetDocumentsPath, "Include/CSS/dash_style_overrides.css"))

	// Scrap remaining docs from www
	scrapDocs(docsetDocumentsPath, db)

	copyFile(filepath.Join(docsetDocumentsPath, "Include", "Images", "site_icon.png"), filepath.Join(docsetPath, "icon.png"))

	// Not sure why we remove this, navigationbar is not included anyway since all script tags are removed
	// mayber required later for toc support
	jsFiles, err := filepath.Glob(filepath.Join(docsetDocumentsPath, "Include/Javascript/navigationBar*.js"))

	if err != nil {
		panic(err)
	}

	for _, f := range jsFiles {
		if err := os.Remove(f); err != nil {
			panic(err)
		}
	}
}

// copy file helper
func copyFile(srcPath string, destPath string) {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		fmt.Println(err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		fmt.Println(err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		fmt.Println(err)
	}
}
