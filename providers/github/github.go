/*
 * Ferret
 * Copyright (c) 2016 Yieldbot, Inc.
 * For the full copyright and license information, please view the LICENSE.txt file.
 */

// Package github implements AnswerHub provider
package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/yieldbot/ferret/search"
)

func init() {
	// Init the provider
	var p = Provider{
		url:        strings.TrimSuffix(os.Getenv("FERRET_GITHUB_URL"), "/"),
		token:      os.Getenv("FERRET_GITHUB_TOKEN"),
		searchUser: os.Getenv("FERRET_GITHUB_SEARCH_USER"),
	}

	// Register the provider
	if err := search.Register("github", &p); err != nil {
		panic(err)
	}
}

// Provider represents the provider
type Provider struct {
	url        string
	token      string
	searchUser string
}

// SearchResult represent the structure of the search result
type SearchResult struct {
	TotalCount        int  `json:"total_count"`
	IncompleteResults bool `json:"incomplete_results"`
	Items             []*SearchResultItems
}

// SearchResultItems represent the structure of the search result items
type SearchResultItems struct {
	Name       string                       `json:"name"`
	Path       string                       `json:"path"`
	HTMLUrl    string                       `json:"html_url"`
	Repository *SearchResultItemsRepository `json:"repository"`
}

// SearchResultItemsRepository represent the structure of the search result items repository
type SearchResultItemsRepository struct {
	Fullname    string `json:"full_name"`
	Description string `json:"description"`
}

// Search makes a search
func (provider *Provider) Search(keyword string) ([]search.ResultItem, error) {

	// Prepare the request
	query := fmt.Sprintf("%s/search/code?q=%s", provider.url, url.QueryEscape(keyword))
	if provider.searchUser != "" {
		query += fmt.Sprintf("+user:%s", url.QueryEscape(provider.searchUser))
	}
	req, err := http.NewRequest("GET", query, nil)
	if provider.token != "" {
		req.Header.Set("Authorization", "token "+provider.token)
	}

	// Make the request
	res, err := provider.do(req)
	if err != nil {
		return nil, errors.New("failed to fetch search result. Error: " + err.Error())
	}

	// Parse and prepare the result
	var sr SearchResult
	if err = json.Unmarshal(res, &sr); err != nil {
		return nil, errors.New("failed to unmarshal JSON data. Error: " + err.Error())
	}
	var result []search.ResultItem
	for _, v := range sr.Items {
		ri := search.ResultItem{
			Description: fmt.Sprintf("%s: %s", v.Repository.Fullname, v.Path),
			Link:        v.HTMLUrl,
		}
		result = append(result, ri)
	}

	return result, nil
}

// do makes request
func (provider *Provider) do(req *http.Request) ([]byte, error) {

	// Do request
	var client = &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// Read data
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// Check response
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return data, errors.New("bad response: " + fmt.Sprintf("%d", res.StatusCode))
	}

	return data, nil
}