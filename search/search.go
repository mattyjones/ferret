/*
 * Ferret
 * Copyright (c) 2016 Yieldbot, Inc.
 * For the full copyright and license information, please view the LICENSE.txt file.
 */

// Package search provides search interface and functionality
package search

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"

	"github.com/yieldbot/gocli"
)

var (
	goCommand = "open"
	providers = make(map[string]Searcher)
)

func init() {
	if os.Getenv("FERRET_GOTO_CMD") != "" {
		goCommand = os.Getenv("FERRET_GOTO_CMD")
	}
}

// Searcher is the interface that must be implemented by a search provider
type Searcher interface {
	// Search makes a search
	Search(keyword string) ([]ResultItem, error)
}

// ResultItem represents a search result item structure
type ResultItem struct {
	Description string
	Link        string
}

// Register registers a search provider
func Register(name string, provider Searcher) error {
	if _, ok := providers[name]; ok {
		return errors.New("provider " + name + " is already registered")
	}
	providers[name] = provider
	return nil
}

// Providers returns a sorted list of the names of the registered providers
func Providers() []string {
	var list = []string{}
	for name := range providers {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

// ByKeyword make a search by the given provider and keyword
func ByKeyword(provider, keyword string, args map[string]string) {

	// Check the provider
	s, ok := providers[provider]
	if !ok {
		log.Fatalf("invalid provider. Possible providers are %s", Providers())
	}

	// Search
	results, err := s.Search(keyword)
	if err != nil {
		log.Fatalf("failed to search due to %s", err.Error())
	}

	// Goto
	if n, ok := args["goto"]; ok {
		i, err := strconv.Atoi(n)
		if err != nil || (i <= 0 || len(results) < i) {
			log.Fatalf("invalid result # to go. It should be between 1 and %d", len(results))
		}
		link := results[i-1].Link
		if _, err = exec.Command(goCommand, link).Output(); err != nil {
			log.Fatalf("failed to go to %s due to %s. Check FERRET_GOTO_CMD environment variable", link, err.Error())
		}
		return
	}

	// Prepare output
	var t = gocli.Table{}
	t.AddRow(1, "#", "TITLE")
	for i, v := range results {
		t.AddRow(i+2, fmt.Sprintf("%d", i+1), v.Description)
	}
	t.PrintData()
}