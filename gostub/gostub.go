package gostub

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/gostub/gostub/models"
)

type Gostub struct {
	Port       string
	OutputPath string
}

func (g *Gostub) Run() {
	http.HandleFunc("/", g.HandleStubRequest)
	http.HandleFunc("/gostub/shutdown", handleShutdown)
	portAddress := ":" + g.Port
	log.Fatal(http.ListenAndServe(portAddress, nil))
}

func (g *Gostub) HandleStubRequest(w http.ResponseWriter, r *http.Request) {
	pathPatternList := g.RecursiveGetFilePath(r.Method)
	requestPath := r.URL.Path
	result, pathParams, matchError := g.MatchRoute(pathPatternList, requestPath)
	if matchError != nil {
		w.WriteHeader(http.StatusNotFound)
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
	reqParam := requestParameter(r)
	fmt.Printf("\nreceive request: [%v] %v\n", r.Method, r.URL.Path)
	fmt.Printf("request parameter: %v\n", reqParam)
	fmt.Printf("path parameter: %v\n", pathParams)
	for _, handler := range list.Handlers {
		if isMatchRequest(r, pathParams, reqParam, handler) {
			fmt.Printf("handle pattern: %+v\n", handler.Content.Body)
			g.SetContent(w, matchPattern, handler.Content)
			return
		}
	}
	fmt.Printf("default pattern: %+v\n", list.Default.Body)
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
				*pathList = append(*pathList, path+f.Name())
			}
			g.recursiveGetFilePath(subPath, method, pathList)
		}
	}
}

func (g *Gostub) RootPath() string {
	if g.OutputPath == "" {
		return "/"
	}
	return fmt.Sprintf("/%v/", g.OutputPath)
}

func (g *Gostub) MatchRoute(pathList []string, requestPath string) (*string, map[string]string, error) {
	if g.OutputPath != "" {
		requestPath = "/" + g.OutputPath + requestPath
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
		bodyFilePath = "/" + g.OutputPath + content.Body
	}
	for k, v := range content.Header {
		w.Header().Add(k, v)
	}
	for k, v := range content.Cookie {
		cookie := &http.Cookie{
			Name:  k,
			Value: v,
		}
		http.SetCookie(w, cookie)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(content.Status)
	response, _ := ioutil.ReadFile("." + bodyFilePath)
	fmt.Fprint(w, string(response))
}

func (g *Gostub) IsMatchRoute(route string, path string) (bool, map[string]string) {
	splitRoute := strings.Split(route, "/")
	splitPath := strings.Split(path, "/")
	params := map[string]string{}
	if len(splitRoute) != len(splitPath) {
		return false, nil
	}
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

func requestParameter(r *http.Request) map[string]string {
	if r.Method == http.MethodPost {
		return postParameter(r.Body)
	} else if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodDelete {
		params := getParameter(r.URL.Query())
		return params
	}
	return map[string]string{}
}

func getParameter(query url.Values) map[string]string {
	var queryParams = map[string]string{}
	for k := range query {
		queryParams[k] = query.Get(k)
	}
	return queryParams
}

func postParameter(b io.ReadCloser) map[string]string {
	var postParamsBox map[string]interface{}
	if err := json.NewDecoder(b).Decode(&postParamsBox); err != nil {
		return map[string]string{}
	}
	var postParams = map[string]string{}
	for k, v := range postParamsBox {
		vs := fmt.Sprint(v)
		postParams[k] = vs
	}
	return postParams
}

func isMatchRequest(request *http.Request, pathParams map[string]string, reqParams map[string]string, handler models.Handler) bool {
	if len(handler.Path)+len(handler.Header)+len(handler.Param) == 0 {
		return false
	}
	for k, v := range handler.Path {
		if !isMatchRegex(fmt.Sprintf("%v", v), pathParams[k]) {
			return false
		}
	}
	for k, v := range handler.Header {
		if !isMatchRegex(fmt.Sprintf("%v", v), request.Header.Get(k)) {
			return false
		}
	}
	for k, v := range handler.Param {
		if !isMatchRegex(fmt.Sprintf("%v", v), reqParams[k]) {
			return false
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
