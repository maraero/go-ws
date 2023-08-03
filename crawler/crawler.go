package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/html"
)

const (
	resDirPath = "./result"
	sourceData = "500.jsonl"
)

type FileLine struct {
	URL        string   `json:"url"`
	Categories []string `json:"categories"`
}

type URLData struct {
	title string
	desc  string
}

func main() {
	prepareResDir(resDirPath)
	readFileByLine(sourceData)
}

func prepareResDir(dirname string) {
	os.RemoveAll(dirname)
	if err := os.Mkdir(dirname, os.ModePerm); err != nil {
		log.Fatalf("cannot create %s directory: %s", dirname, err.Error())
	}
}

func readFileByLine(path string) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("cannot open file %s: %s", path, err.Error())
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fileline, err := extractInfoFromFileLine(line)
		if err == nil {
			fmt.Println(fileline)
			processFileLine(fileline)
		} else {
			fmt.Printf("scan error: %s\n", err.Error())
		}
	}
}

func extractInfoFromFileLine(line string) (*FileLine, error) {
	var fl *FileLine

	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil, errors.New("empty string")
	}

	err := json.Unmarshal([]byte(trimmed), &fl)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal line\"%s\": %s", trimmed, err)
	}

	if len(fl.Categories) == 0 {
		return nil, fmt.Errorf("empty categories in line \"%s\"", trimmed)
	}

	return fl, nil
}

func processFileLine(fl *FileLine) {
	fetchedData, err := processURL(fl.URL)
	if err != nil {
		// ...
	} else {
		fmt.Println(fetchedData)
	}
}

func processURL(url string) (*URLData, error) {
	resp, err := http.Get(url)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}
	return extractDataFromHTTPResp(resp)
}

func extractDataFromHTTPResp(resp *http.Response) (*URLData, error) {
	var r URLData
	z := html.NewTokenizer(resp.Body)

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			return nil, errors.New("cannot parse html")
		}

		t := z.Token()

		// <title> not </title>
		if t.Type == html.StartTagToken && t.Data == "title" {
			if z.Next() == html.TextToken {
				r.title = strings.TrimSpace(z.Token().Data)
			}
			continue
		}

		// can be <meta> or <meta />
		if (t.Type == html.SelfClosingTagToken || t.Type == html.StartTagToken) && t.Data == "meta" {
			desc, err := getDescFromAttribute(t)
			if err != nil {
				continue
			}
			r.desc = desc
		}

		// stop parsing on </head>
		if t.Type == html.EndTagToken && t.Data == "head" {
			break
		}
	}

	return &r, nil
}

func getDescFromAttribute(t html.Token) (string, error) {
	attrs := make(map[string]string)

	for _, attr := range t.Attr {
		attrs[attr.Key] = attr.Val
	}

	name := attrs["name"]
	content, contentOK := attrs["content"]

	if name == "description" && contentOK {
		return strings.TrimSpace(content), nil
	}

	return "", errors.New("no description")
}
