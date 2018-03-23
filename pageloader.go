// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "io/ioutil"

// Page ... super !
type Page struct {
	Title string
	Body  []byte
}

func (p *Page) save() error {
	filename := pathToDatas + p.Title
	return ioutil.WriteFile(filename, p.Body, 0600)
}

// LoadPage ... ca load la page
func LoadPage(title string) (*Page, error) {
	filename := pathToDatas + title
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return &Page{Title: "Error Page", Body: []byte("Page couldnt be loaded")}, err
	}
	return &Page{Title: title, Body: body}, nil
}
