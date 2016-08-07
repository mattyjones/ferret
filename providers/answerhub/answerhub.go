/*
 * Ferret
 * Copyright (c) 2016 Yieldbot, Inc.
 * For the full copyright and license information, please view the LICENSE.txt file.
 */

// Package answerhub implements AnswerHub provider
package answerhub

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"
)

// Provider represents the provider
type Provider struct {
	enabled  bool
	name     string
	title    string
	priority int64
	url      string
	username string
	password string
}

// Register registers the provider
func Register(f func(provider interface{}) error) {
	var p = Provider{
		name:     "answerhub",
		title:    "AnswerHub",
		priority: 1000,
		url:      strings.TrimSuffix(os.Getenv("FERRET_ANSWERHUB_URL"), "/"),
		username: os.Getenv("FERRET_ANSWERHUB_USERNAME"),
		password: os.Getenv("FERRET_ANSWERHUB_PASSWORD"),
	}
	if p.url != "" {
		p.enabled = true
	}

	if err := f(&p); err != nil {
		panic(err)
	}
}

// SearchResult represents the structure of the search result
type SearchResult struct {
	List []*SRList `json:"list"`
}

// SRList represents the structure of the search result list
type SRList struct {
	ID           int        `json:"id"`
	Title        string     `json:"title"`
	Body         string     `json:"body"`
	Author       *SRLAuthor `json:"author"`
	CreationDate int64      `json:"creationDate"`
}

// SRLAuthor represents the structure of the search result list author field
type SRLAuthor struct {
	Username string `json:"username"`
	Realname string `json:"realname"`
}

// Search makes a search
func (provider *Provider) Search(ctx context.Context, args map[string]interface{}) ([]map[string]interface{}, error) {

	results := []map[string]interface{}{}
	page, ok := args["page"].(int)
	if page < 1 || !ok {
		page = 1
	}
	limit, ok := args["limit"].(int)
	if limit < 1 || !ok {
		limit = 10
	}
	keyword, ok := args["keyword"].(string)

	u := fmt.Sprintf("%s/services/v2/node.json?page=%d&pageSize=%d&q=%s*", provider.url, page, limit, url.QueryEscape(keyword))
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, errors.New("failed to prepare request. Error: " + err.Error())
	}
	if provider.username != "" || provider.password != "" {
		req.SetBasicAuth(provider.username, provider.password)
	}

	res, err := ctxhttp.Do(ctx, nil, req)
	if err != nil {
		return nil, err
	} else if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, errors.New("bad response: " + fmt.Sprintf("%d", res.StatusCode))
	}
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var sr SearchResult
	if err := json.Unmarshal(data, &sr); err != nil {
		return nil, errors.New("failed to unmarshal JSON data. Error: " + err.Error())
	}
	for _, v := range sr.List {
		d := strings.TrimSpace(v.Body)
		if len(d) > 255 {
			d = d[0:252] + "..."
		} else if len(d) == 0 {
			if v.Author.Realname != "" {
				d = "Asked by " + v.Author.Realname
			} else {
				d = "Asked by " + v.Author.Username
			}
		}
		ri := map[string]interface{}{
			"Link":        fmt.Sprintf("%s/questions/%d/", provider.url, v.ID),
			"Title":       v.Title,
			"Description": d,
			"Date":        time.Unix(0, v.CreationDate*int64(time.Millisecond)),
		}
		results = append(results, ri)
	}

	return results, err
}
