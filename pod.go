package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
)

type Query struct {
	Podcast Channel `xml:"channel"`
}

type Channel struct {
	Title       string    `xml:"title"`
	Desc        string    `xml:"description"`
	NewFeedUrl  []FeedUrl `xml:"link"`
	EpisodeList []Episode `xml:"item"`
}

type Episode struct {
	Title     string     `xml:"title"`
	Desc      string     `xml:"description"`
	PubDate   string     `xml:"pubDate"`
	Enclosure EpisodeUrl `xml:"enclosure"`
}

type FeedUrl struct {
	Rel  string `xml:"rel,attr"`
	Link string `xml:"href,attr"`
}

type EpisodeUrl struct {
	Link string `xml:"url,attr"`
}

func die(status int) {
	if status > 0 {
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}

func usage(status int) {
	fmt.Print(`pod - small podcast thing

	pod command arguments

available commands are
	list			list all available podcast
	info PODCAST		print info about PODCAST
	refresh PODCAST		refresh the PODCAST
	fetch PODCAST		get the latest episode of PODCAST
	pull			get the latest episode of all podcasts
	clean PODCAST		remove media files for PODCAST
	help			display this help
`)
	die(status)
}

func fetch(url, fileName string) {
	fmt.Printf("%-10s %s\n", "fetching", url)

	output, err := os.Create(fileName)
	if err != nil {
		fmt.Println(err)
		die(1)
	}
	defer output.Close()

	response, err := http.Get(url)
	if err != nil {
		fmt.Println("error while downloading", url, "\n", err)
		die(1)
	}
	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)
	output.Sync()

	fmt.Printf("%-10s %s %v bytes\n", "fetched", url, n)
	return
}

func list() {
	files, err := ioutil.ReadDir("rss")
	if err != nil {
		fmt.Println(err)
		die(1)
	}

	for _, file := range files {
		fmt.Printf("%v\n", file.Name())
	}
	die(0)
}

func podInfo(filename string) {
	xmlFile, err := os.Open(filename)
	if err != nil {
		fmt.Printf("%v\n", err)
		die(1)
	}

	defer xmlFile.Close()

	file, _ := ioutil.ReadAll(xmlFile)

	q := Query{}
	//q.
	//err = xml.Unmarshal(file, &q)
	d := xml.NewDecoder(bytes.NewReader(file))
	err = d.Decode(&q)
	c := q.Podcast

	fmt.Printf("title\t\t%v\n", c.Title)

	for _, feedurl := range c.NewFeedUrl {
		if feedurl.Rel == "self" {
			fmt.Printf("url\t\t%v\n", feedurl.Link)
		}
	}
	fmt.Printf("desc\t\t%v\n", c.Desc)
	lastEpisode := c.EpisodeList[0]
	fmt.Printf("last episode\t%v\n\t\t%v\n", lastEpisode.PubDate, lastEpisode.Title)
	die(0)
	/*
		for _, episode := range c.EpisodeList {
			fmt.Printf("\t%s\n", episode)
		}
	*/
}

func fetchPodcast(podid string) {
	xmlFile, err := os.Open(podid)
	if err != nil {
		fmt.Printf("%v\n", err)
		die(1)
	}

	file, _ := ioutil.ReadAll(xmlFile)

	q := Query{}

	d := xml.NewDecoder(bytes.NewReader(file))
	err = d.Decode(&q)
	c := q.Podcast

	var url string
	for _, i := range c.NewFeedUrl {
		if i.Rel == "self" {
			url = i.Link
			fetch(url, "tmp")
			break
		}
	}
	if url == "" {
		fmt.Printf("no update link found\n")
		die(1)
	}

	err = xmlFile.Sync()
	if err != nil {
		fmt.Printf("%v\n", err)
		die(1)
	}
	err = xmlFile.Close()
	if err != nil {
		fmt.Printf("%v\n", err)
		die(1)
	}

	err = os.Remove(podid)
	if err != nil {
		fmt.Printf("%v\n", err)
		die(1)
	}

	err = os.Rename("tmp", podid)
	if err != nil {
		fmt.Printf("%v\n", err)
		die(1)
	}

	return
	/*
		for _, episode := range c.EpisodeList {
			fmt.Printf("\t%s\n", episode)
		}
	*/
}

func fetchEpisode(podid string) {
	check(podid)
	xmlFile, err := os.Open(podid)
	if err != nil {
		fmt.Printf("%v\n", err)
		die(1)
	}

	defer xmlFile.Close()

	file, _ := ioutil.ReadAll(xmlFile)

	q := Query{}

	d := xml.NewDecoder(bytes.NewReader(file))
	err = d.Decode(&q)
	c := q.Podcast

	url := c.EpisodeList[0].Enclosure.Link
	podname := path.Base(podid)
	filename := "media/" + podname + "/" + path.Base(url)

	//check if the file to download already exists
	_, err = os.Stat(filename)
	if err == nil {
		fmt.Fprintf(os.Stdout, "media already downloaded\n")
		return
	}
	fetch(url, filename)
	return
	/*
		for _, episode := range c.EpisodeList {
			fmt.Printf("\t%s\n", episode)
		}
	*/
}

func pull() {
	files, err := ioutil.ReadDir("rss")
	if err != nil {
		fmt.Println(err)
		die(1)
	}

	for _, file := range files {
		check(file.Name())
		fetchPodcast("rss/" + file.Name())
		fmt.Printf("fetching %v\n", file.Name())
		fetchEpisode("rss/" + file.Name())
	}
	die(0)
}

func clean(mediaid string) {
	if mediaid == "all" {
		cleanall()
	}
	err := os.Remove(mediaid)
	if err != nil {
		fmt.Println(err)
		die(1)
	}
	fmt.Printf("cleaning done\n")
	die(0)
}

func cleanall() {
	err := os.RemoveAll("media/")
	if err != nil {
		fmt.Println(err)
		die(1)
	}
	fmt.Printf("cleaning done\n")
	die(0)
}

func check(podid string) {
	_, err := os.Stat("media")
	if os.IsNotExist(err) {
		fmt.Printf("media doesn't exist, creating dir\n")
		err = os.Mkdir("media", 0755)
		if err != nil {
			fmt.Println(err)
			die(1)
		}
	}
	_, err = os.Stat("media/" + path.Base(podid))
	if os.IsNotExist(err) {
		fmt.Printf("media/%v doesn't exist, creating dir\n", path.Base(podid))
		err = os.Mkdir("media/"+path.Base(podid), 0755)
		if err != nil {
			fmt.Println(err)
			die(1)
		}
	}
	return
}

func main() {

	if len(os.Args) < 2 {
		fmt.Println("not enough arguments!")
		usage(1)
	}

	switch os.Args[1] {
	case "list":
		list()
	case "info":
		if len(os.Args) < 3 {
			fmt.Println("not enough arguments!")
			die(1)
		}
		inputFile := os.Args[2]
		podInfo(inputFile)
	case "fetch":
		if len(os.Args) < 3 {
			fmt.Println("not enough arguments!")
			die(1)
		}
		fetchPodcast(os.Args[2])
		fetchEpisode(os.Args[2])
		die(0)
	case "pull":
		pull()
	case "clean":
		if len(os.Args) < 3 {
			fmt.Println("not enough arguments!")
			die(1)
		}
		clean(os.Args[2])
	case "refresh":
		if len(os.Args) < 3 {
			fmt.Println("not enough arguments!")
			die(1)
		}
		fetchPodcast(os.Args[2])
	case "help":
		usage(0)
	default:
		usage(1)
	}

	return
}
