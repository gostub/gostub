package main

import (
	"fmt"
	"log"
	"net/http"
	"io/ioutil"
	"os"
	"regexp"
	"encoding/json"
	"strings"
	"errors"
	"github.com/gostub/gostub/models"
)

type Gostub struct {
	port string
	outputPath string
}

func (g *Gostub) Run() {
	http.HandleFunc("/", g.HandleStubRequest)
	http.HandleFunc("/gostub/shutdown", handleShutdown)
	portAddress := ":" + g.port
	log.Fatal(http.ListenAndServe(portAddress, nil))
}

func (g *Gostub) HandleStubRequest(w http.ResponseWriter, r *http.Request) {
	pathPatternList := g.RecursiveGetFilePath(r.Method)
	requestPath := r.URL.Path
	result, params ,matchError := g.MatchRoute(pathPatternList, requestPath)
	if matchError != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Not found path content (%v)", requestPath)
		return
	}
	matchPattern := *result
	contentPath := matchPattern + "/$" + strings.ToUpper(r.Method) + ".json"
	content, readError := ioutil.ReadFile("." + contentPath)
	if readError != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid path content (%v)", contentPath)
		return
	}
	list := new(models.ContentList)
	json.Unmarshal(content, &list)
	for _, handler := range list.Handlers {
		if isMatchRequest(r, params, handler) {
			fmt.Println("handler request match")
			g.SetContent(w, matchPattern, handler.Content)
			return
		}
	}
	g.SetContent(w, matchPattern, list.Default)
}

func (g *Gostub) RecursiveGetFilePath(method string) []string {
	var pathPatternList []string
	g.recursiveGetFilePath(g.RootPath(), method, &pathPatternList)
	return pathPatternList
}

func (g *Gostub) recursiveGetFilePath(path string, method string, pathList *[]string) {
	files, _ := ioutil.ReadDir("." + path)
	for _, f := range files {
		if f.IsDir() {
			subPath := path + f.Name() + "/"
			if exists(subPath + "$" + method + ".json") {
				*pathList = append(*pathList, path + f.Name())
			}
			g.recursiveGetFilePath(subPath, method, pathList)
		}
	}
}

func (g *Gostub) RootPath() string {
	if g.outputPath == "" {
		return "/"
	}
	return fmt.Sprintf("/%v/", g.outputPath)
}

func (g *Gostub) MatchRoute(pathList []string, requestPath string) (*string, map[string]string, error) {
	if g.outputPath != "" {
		requestPath = "/" + g.outputPath + requestPath
	}
	var filteredPathPatternList []string
	var filteredPathParameters []map[string]string
	for _, path := range pathList {
		ret, params := g.IsMatchRoute(path, requestPath)
		if ret {
			filteredPathPatternList = append(filteredPathPatternList, path)
			filteredPathParameters = append(filteredPathParameters, params)
		}
	}
	if len(filteredPathPatternList) == 0 {
		return nil, nil, errors.New("not found route")
	}
	// FIXME とりあえず一番最後のpathを指定
	n := len(filteredPathPatternList)
	return &filteredPathPatternList[n-1], filteredPathParameters[n-1], nil
}

func (g *Gostub) SetContent(w http.ResponseWriter, pattern string, content models.Content) {
	bodyFilePath := pattern + "/" + content.Body
	if strings.HasPrefix(content.Body, "/") {
		bodyFilePath = "/" + g.outputPath + content.Body
	}
	for k, v := range content.Header {
		w.Header().Add(k, v)
	}
	for k, v := range content.Cookie {
		cookie := &http.Cookie{
			Name: k,
			Value: v,
		}
		http.SetCookie(w, cookie)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(content.Status)
	response, _ := ioutil.ReadFile("." + bodyFilePath)
	fmt.Fprint(w, string(response))
}

func (g *Gostub) IsMatchRoute(route string, path string) (bool, map[string]string) {
	splitRoute := strings.Split(route, "/")
	splitPath := strings.Split(path, "/")
	params := map[string]string{}
	for idx, pathNode := range splitPath {
		if len(splitRoute)-1 < idx {
			return false, nil
		}
		routeNode := splitRoute[idx]
		if routeNode != pathNode && !strings.HasPrefix(routeNode, ":") {
			return false, nil
		}
		if strings.HasPrefix(routeNode, ":") {
			params[routeNode[1:]] = pathNode
		}
	}
	return true, params
}

func handleShutdown(w http.ResponseWriter, r *http.Request) {
	log.Fatal("Stop gostub server.")
}

func isMatchRequest(request *http.Request, params map[string]string, handler models.Handler) bool {
	for k ,v := range handler.Path {
		if !isMatchRegex(v, params[k]) {
			return false
		}
	}
	for k ,v := range handler.Header {
		if !isMatchRegex(v, request.Header.Get(k)) {
			return false
		}
	}
	for k ,v := range handler.Param {
		if request.Method == http.MethodGet || request.Method == http.MethodHead || request.Method == http.MethodDelete {
			if !isMatchRegex(v, request.URL.Query().Get(k)) {
				return false
			}
		} else if request.Method == http.MethodPost {
			if !isMatchRegex(v, request.FormValue(k)) {
				return false
			}
		}
	}
	return true
}

func exists(filename string) bool {
	_, err := os.Stat("." + filename)
	return err == nil
}

func isMatchRegex(regexPattern string, target string) bool {
	regex := regexp.MustCompile(regexPattern)
	return regex.MatchString(target)
}
