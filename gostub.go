package gostub

import (
	"fmt"
	"log"
	"net/http"
	"io/ioutil"
	"os"
	"regexp"
	"encoding/json"
	"strings"
)

type ContentList struct {
	Default 	Content		`json:"default"`
	Handlers    []Content  	`json:"handlers"`
}

type Content struct {
	Response 	string 				`json:"response"`
	Status 		int 				`json:"status"`
	Header 		map[string]string 	`json:"header"`
	Param 		map[string]string 	`json:"param"`
}

func (c Content) isMatchRequest(r *http.Request) bool {
	for k ,v := range c.Header {
		if !isMatchRegex(v, r.Header.Get(k)) {
			return false
		}
	}
	for k ,v := range c.Param {
		if r.Method == http.MethodGet {
			if !isMatchRegex(v, r.URL.Query().Get(k)) {
				return false
			}
		} else if r.Method == http.MethodPost {
			if !isMatchRegex(v, r.PostForm.Get(k)) {
				return false
			}
		}
	}
	return true
}

func Run() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var pathPatternList []string
		recursiveGetFilePath("/", r.Method, &pathPatternList)
		requestPath := r.URL.Path
		filteredPathPatternList := filtered(pathPatternList, func(p string) bool {
			return isMatchRegex("^" + p + "$", requestPath)
		})
		if len(filteredPathPatternList) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "Not found path pattern")
			return
		}
		n := len(filteredPathPatternList)
		pathPattern := filteredPathPatternList[n-1] // FIXME とりあえず一番最後のpathを指定
		contentPath := pathPattern + "/$" + strings.ToUpper(r.Method) + ".json"

		if !exists(contentPath) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Not found path content (%v)", contentPath)
			return
		}
		content, readError := ioutil.ReadFile("." + contentPath)
		if readError != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Invalid path content (%v)", contentPath)
			return
		}
		list := new(ContentList)
		json.Unmarshal(content, &list)
		w.Header().Set("Content-Type", "application/json")
		for _, handler := range list.Handlers {
			if handler.isMatchRequest(r) {
				w.WriteHeader(handler.Status)
				// TODO: handler.Responseが '/' から始まっていたらrootを指す
				response, _ := ioutil.ReadFile("." + pathPattern + "/" + handler.Response)
				fmt.Fprint(w, string(response))
				return
			}
		}
		w.WriteHeader(list.Default.Status)
		response, _ := ioutil.ReadFile("." + pathPattern + "/" + list.Default.Response)
		fmt.Fprint(w, string(response))
	})
	log.Fatal(http.ListenAndServe(":8181", nil))
}

func main() {
	Run()
}

func recursiveGetFilePath(path string, method string, contentPaths *[]string) {
	files, _ := ioutil.ReadDir("." + path)
	for _, f := range files {
		if f.IsDir() {
			subPath := path + f.Name() + "/"
			if exists(subPath + "$" + method + ".json") {
				*contentPaths = append(*contentPaths, path + f.Name())
			}
			recursiveGetFilePath(subPath, method, contentPaths)
		}
	}
}

func filtered(strings []string, filter func(string,) bool) []string {
	var res []string
	for _, path := range strings {
		if filter(path) {
			res = append(res, path)
		}
	}
	return res
}

func exists(filename string) bool {
	_, err := os.Stat("." + filename)
	return err == nil
}

func isMatchRegex(regexPattern string, target string) bool {
	regex := regexp.MustCompile(regexPattern)
	return regex.MatchString(target)
}
