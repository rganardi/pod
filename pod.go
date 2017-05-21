package main

import (
	"fmt"
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
	EpisodeList	[]Episode	`xml:"item"`
}

type Episode struct {
	Title	string `xml:"title"`
	Desc	string `xml:"description"`
	PubDate	string `xml:"pubDate"`
	Enclosure	EpisodeUrl `xml:"enclosure"`
}

type EpisodeUrl struct {
	Link	string `xml:"url,attr"`
}
/*
func (c Channel) String() string {
	return fmt.Sprintf("%s - %d hi there", c.Title, c.Desc)
}

func (e Episode) String() string {
	return fmt.Sprintf("%s - %s", e.Title, e.Desc)
}
*/

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
	err = xml.Unmarshal(file, &q)

	c := q.Podcast
	fmt.Printf("%s - %s\n", c.Title, c.Desc)
	for _, episode := range c.EpisodeList {
		fmt.Printf("\t%s\n", episode)
	}

	return
}
