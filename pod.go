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
	fmt.Fprintf(os.Stdout, "%-10s %s\n", "fetching", url)

	output, err := os.Create(fileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		die(1)
	}
	defer output.Close()

	response, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while downloading %v\n%v\n", url, err)
		die(1)
	}
	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)
	output.Sync()

	fmt.Fprintf(os.Stdout, "%-10s %s %v bytes\n", "fetched", url, n)
	return
}

func list() {
	files, err := ioutil.ReadDir("rss")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		die(1)
	}

	for _, file := range files {
		fmt.Fprintf(os.Stdout, "%v\n", file.Name())
	}
	die(0)
}

func podInfo(filename string) {
	xmlFile, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
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

	fmt.Fprintf(os.Stdout, "title\t\t%v\n", c.Title)

	for _, feedurl := range c.NewFeedUrl {
		if feedurl.Rel == "self" {
			fmt.Fprintf(os.Stdout, "url\t\t%v\n", feedurl.Link)
		}
	}
	fmt.Fprintf(os.Stdout, "desc\t\t%v\n", c.Desc)
	lastEpisode := c.EpisodeList[0]
	fmt.Fprintf(os.Stdout, "last episode\t%v\n\t\t%v\n", lastEpisode.PubDate, lastEpisode.Title)
	die(0)
	/*
		for _, episode := range c.EpisodeList {
			fmt.Printf("\t%s\n", episode)
		}
	*/
}

func fetchPodcast(podid string) error {
	xmlFile, err := os.Open(podid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
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
		fmt.Fprintf(os.Stderr, "%s no update link found\n", podid)
		return fmt.Errorf("%s no update link found\n", podid)
	}

	err = xmlFile.Sync()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		die(1)
	}
	err = xmlFile.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		die(1)
	}

	err = os.Remove(podid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		die(1)
	}

	err = os.Rename("tmp", podid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		die(1)
	}

	return nil
	/*
		for _, episode := range c.EpisodeList {
			fmt.Printf("\t%s\n", episode)
		}
	*/
}

func fetchEpisode(podid string) {
	check(podid)
	podname := path.Base(podid)
	xmlFile, err := os.Open(podid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		die(1)
	}

	defer xmlFile.Close()

	file, _ := ioutil.ReadAll(xmlFile)

	q := Query{}

	d := xml.NewDecoder(bytes.NewReader(file))
	err = d.Decode(&q)
	c := q.Podcast

	url := c.EpisodeList[0].Enclosure.Link
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
		fmt.Fprintf(os.Stderr, "%v\n", err)
		die(1)
	}

	for _, file := range files {
		check(file.Name())
		err = fetchPodcast("rss/" + file.Name())
		if err != nil {
			//don't download the episode
			continue
		}
		fmt.Fprintf(os.Stdout, "fetching %v\n", file.Name())
		fetchEpisode("rss/" + file.Name())
	}
	die(0)
}

func clean(podid string) {
	if podid == "all" {
		cleanall()
		return
	}

	files, err := ioutil.ReadDir("media/" + path.Base(podid))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		die(1)
	}

	xmlFile, err := os.Open(podid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		die(1)
	}
	defer xmlFile.Close()

	fd, _ := ioutil.ReadAll(xmlFile)

	q := Query{}

	d := xml.NewDecoder(bytes.NewReader(fd))
	err = d.Decode(&q)
	c := q.Podcast

	//find the latest episode
	var latest string
	for i := 0; i < len(c.EpisodeList) && latest == ""; i++ {
		filename := path.Base(c.EpisodeList[i].Enclosure.Link)
		for _, file := range files {
			if file.Name() == filename {
				latest = filename
				break
			}
		}
	}
	if latest == "" {
		fmt.Fprintf(os.Stderr, "latest episode not found, aborting clean\n")
		die(1)
	}

	for _, file := range files {
		if file.Name() != latest {
			err = os.Remove("media/" + path.Base(podid) + "/" + file.Name())
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				die(1)
			}
		}
	}

	fmt.Fprintf(os.Stdout, "cleaning done\n")
	return
}

func cleanall() {
	pods, err := ioutil.ReadDir("media/")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		die(1)
	}

	for _, pod := range pods {
		clean("rss/" + pod.Name())
	}

	fmt.Fprintf(os.Stdout, "cleaning done\n")
	return
}

func check(podid string) {
	_, err := os.Stat("media")
	if os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "media doesn't exist, creating dir\n")
		err = os.Mkdir("media", 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			die(1)
		}
	}
	_, err = os.Stat("media/" + path.Base(podid))
	if os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "media/%v doesn't exist, creating dir\n", path.Base(podid))
		err = os.Mkdir("media/"+path.Base(podid), 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			die(1)
		}
	}
	return
}

func main() {

	err := os.Chdir(os.Getenv("HOME") + "/pod/")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}

	if len(os.Args) < 2 {
		usage(0)
	}

	switch os.Args[1] {
	case "list":
		list()
	case "info":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "not enough arguments!\n")
			die(1)
		}
		inputFile := os.Args[2]
		podInfo(inputFile)
	case "fetch":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "not enough arguments!\n")
			die(1)
		}
		for _, podid := range os.Args[2:] {
			err := fetchPodcast(podid)
			if err == nil {
				fetchEpisode(podid)
			}
		}
		die(0)
	case "pull":
		pull()
	case "clean":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "not enough arguments!\n")
			die(1)
		}
		clean(os.Args[2])
		die(0)
	case "refresh":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "not enough arguments!\n")
			die(1)
		}
		for _, podid := range os.Args[2:] {
			fetchPodcast(podid)
		}
	case "help":
		usage(0)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command\n")
		usage(1)
	}
	return
}
