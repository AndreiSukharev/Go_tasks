package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

func getError(err error) {
	if err != nil {
		panic(err)
	}
}

func getSize(file os.FileInfo) string {
	var pic string
	size := file.Size()
	if size == 0 {
		pic = " (empty)"
	} else {
		pic = fmt.Sprintf(" (%db)", size)
	}
	return pic
}

func reduceFiles(files []os.FileInfo) []os.FileInfo {
	var newFiles []os.FileInfo
	for _, file := range files {
		if file.IsDir() {
			newFiles = append(newFiles, file)
		}
	}
	return newFiles
}

func printDir(out io.Writer, path string, printFiles bool, pic string) error {
	vertLine := "│"
	tab := "\t"
	obj := "├───"
	endObj := "└───"
	var picForPrint string
	var resPic string

	files, err := ioutil.ReadDir(path)
	if !printFiles {
		files = reduceFiles(files)
	}
	lastObj := len(files) - 1

	for i, file := range files {
		filneName := file.Name()
		isDir := file.IsDir()
		newPath := path + "/" + filneName

		if i == lastObj {
			resPic = pic + tab
			picForPrint = pic + endObj + filneName
		} else {
			resPic = pic + vertLine + tab
			picForPrint = pic + obj + filneName
		}

		if !isDir && printFiles {
			size := getSize(file)
			picForPrint += size
		}

		if printFiles || isDir {
			fmt.Fprintln(out, picForPrint)
		}

		if isDir {
			printDir(out, newPath, printFiles, resPic)
		}

	}
	return err
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	err := printDir(out, path, printFiles, "")
	return err
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	getError(err)
}
