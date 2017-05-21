package main

import (
	"fmt"
	"bytes"
	"encoding/xml"
	"os"
	"io/ioutil"
)

type Query struct {
	Podcast 	Channel		`xml:"channel"`
}


type Channel struct {
	Title	string `xml:"title"`
	Desc	string `xml:"description"`
	NewFeedUrl	[]FeedUrl	`xml:"link"`
	EpisodeList	[]Episode	`xml:"item"`
}

type Episode struct {
	Title	string `xml:"title"`
	Desc	string `xml:"description"`
	PubDate	string `xml:"pubDate"`
	Enclosure	EpisodeUrl `xml:"enclosure"`
}

type FeedUrl struct {
	Rel	string `xml:"rel,attr"`
	Link	string `xml:"href,attr"`
}

type EpisodeUrl struct {
	Link	string `xml:"url,attr"`
}


func main() {
	inputFile := os.Args[1]
	fmt.Printf("hi!\n")

	xmlFile, err := os.Open(inputFile)
	if err != nil {
		fmt.Printf("can't open file %T\n", err)
		return
	}

	defer xmlFile.Close()

	file, _ := ioutil.ReadAll(xmlFile)

	q := Query{}
	//q.
	//err = xml.Unmarshal(file, &q)
	d := xml.NewDecoder(bytes.NewReader(file))
	err = d.Decode(&q)
	c := q.Podcast

	fmt.Printf("%v\t%v\n", c.Title, c.Desc)

	for _, feedurl := range c.NewFeedUrl {
		if feedurl.Rel == "self" {
			fmt.Printf("%v\n", feedurl.Link)
		}
	}

	for _, episode := range c.EpisodeList {
		fmt.Printf("\t%s\n", episode)
	}

	return
}
