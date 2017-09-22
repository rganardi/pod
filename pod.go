package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
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

//these are defined in the makefile
var (
	version_number, build_date string = "unknown", "unknown"
	status int = 0
)

func die() {
	if status > 0 {
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}

func usage() {
	defer die()
	fmt.Print("pod - small podcast thing\n")
	fmt.Printf("build %s, %s\n", version_number, build_date)
	fmt.Print(`
	pod command arguments

available commands are
	list			list all available podcast
	info PODCAST		print info about PODCAST
	refresh PODCAST		refresh the PODCAST
	fetch PODCAST		get the latest episode of PODCAST
	pull			get the latest episode of all podcasts
	clean PODCAST		remove media files for PODCAST
	episode PODCAST		see all episodes of PODCAST
	help			display this help
`)
	return
}

func fetch(url, fileName string) {
	//fmt.Fprintf(os.Stdout, "%-10s %s\n", "fetching", url)

	output, err := os.Create(fileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		status = 1
		return
	}
	defer output.Close()

	response, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while downloading %v\n%v\n", url, err)
		status = 1
		return
	}
	defer response.Body.Close()

	_, err = io.Copy(output, response.Body)
	output.Sync()

	//fmt.Fprintf(os.Stdout, "%-10s %s %v bytes\n", "fetched", url, n)
	return
}

func list() {
	defer die()

	files, err := ioutil.ReadDir("rss")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		status = 1
		return
	}

	for _, file := range files {
		fmt.Fprintf(os.Stdout, "%v\n", file.Name())
	}
	return
}

func podInfo(filename string) {
	defer die()

	xmlFile, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		status = 1
		return
	}

	defer xmlFile.Close()

	file, _ := ioutil.ReadAll(xmlFile)

	q := Query{}
	//q.
	//err = xml.Unmarshal(file, &q)
	d := xml.NewDecoder(bytes.NewReader(file))
	err = d.Decode(&q)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while decoding %s\n", filename)
		fmt.Fprintf(os.Stderr, "%v\n", err)
		status = 1
		return
	}
	c := q.Podcast

	fmt.Fprintf(os.Stdout, "title\t\t%v\n", c.Title)

	for _, feedurl := range c.NewFeedUrl {
		if feedurl.Rel == "self" {
			fmt.Fprintf(os.Stdout, "url\t\t%v\n", feedurl.Link)
		}
	}
	//fmt.Fprintf(os.Stdout, "desc\t\t%v\n", c.Desc)
	lastEpisode := c.EpisodeList[0]
	fmt.Fprintf(os.Stdout, "last episode\t%v\n\t\t%v\n", lastEpisode.PubDate, lastEpisode.Title)
	fmt.Fprintf(os.Stdout, "desc\t\t%v\n", lastEpisode.Desc)
	return
	/*
		for _, episode := range c.EpisodeList {
			fmt.Printf("\t%s\n", episode)
		}
	*/
}

func podEpisode(filename string) {
	defer die()

	xmlFile, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		status = 1
		return
	}

	defer xmlFile.Close()

	file, _ := ioutil.ReadAll(xmlFile)

	q := Query{}
	//q.
	//err = xml.Unmarshal(file, &q)
	d := xml.NewDecoder(bytes.NewReader(file))
	err = d.Decode(&q)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while decoding %s\n", filename)
		fmt.Fprintf(os.Stderr, "%v\n", err)
		status = 1
		return
	}
	c := q.Podcast

	env := os.Environ()
	if err != nil {
		fmt.Fprint(os.Stderr, "error getting environment variables\n")
		fmt.Fprintf(os.Stderr, "%v\n", err)
		status = 1
		return
	}

	pager := "/usr/bin/less"
	for _, variable := range env {
		if strings.HasPrefix(variable, "PAGER") {
			pager = strings.TrimPrefix(variable, "PAGER=")
		}
	}

	commandToRun := exec.Command(pager)
	commandToRun.Stdout = os.Stdout
	pagerStdin, err := commandToRun.StdinPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		status = 1
		return
	}

	err = commandToRun.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		status = 1
		return
	}

	for _, episode := range c.EpisodeList {
		fmt.Fprintf(pagerStdin, "episode title\t%v\n", episode.Title)
		fmt.Fprintf(pagerStdin, "date\t\t%v\n", episode.PubDate)
		fmt.Fprintf(pagerStdin, "desc\t\t%v\n", episode.Desc)
		fmt.Fprintf(pagerStdin, "\n")
	}

	pagerStdin.Close()

	err = commandToRun.Wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		status = 1
		return
	}

	return
}
func fetchPodcast(podid string) error {
	xmlFile, err := os.Open(podid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		status = 1
		return err
	}

	defer xmlFile.Close()

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
		status = 1
		return fmt.Errorf("%s no update link found\n", podid)
	}

	err = xmlFile.Sync()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		status = 1
		return err
	}
	err = xmlFile.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		status = 1
		return err
	}

	err = os.Remove(podid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		status = 1
		return err
	}

	err = os.Rename("tmp", podid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		status = 1
		return err
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
		status = 1
		return
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
		//fmt.Fprintf(os.Stdout, "media already downloaded\n")
		return
	}
	fmt.Fprintf(os.Stdout, "%s %-20s %s %-20s\r", "fetching", podname, "eps", c.EpisodeList[0].Title)
	fetch(url, filename)
	return
	/*
		for _, episode := range c.EpisodeList {
			fmt.Printf("\t%s\n", episode)
		}
	*/
}

func pull() {
	defer die()

	files, err := ioutil.ReadDir("rss")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		status = 1
		return
	}

	for _, file := range files {
		check(file.Name())
		fmt.Fprintf(os.Stdout, "%s %-50s\r", "fetching", file.Name())
		err = fetchPodcast("rss/" + file.Name())
		if err != nil {
			//don't download the episode
			continue
		}
		fetchEpisode("rss/" + file.Name())
	}
	return
}

func clean(podid string) {
	files, err := ioutil.ReadDir("media/" + path.Base(podid))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		status = 1
		return
	}

	xmlFile, err := os.Open(podid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		status = 1
		return
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
		status = 1
		return
	}

	for _, file := range files {
		if file.Name() != latest {
			err = os.Remove("media/" + path.Base(podid) + "/" + file.Name())
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				status = 1
				return
			}
		}
	}

	fmt.Fprintf(os.Stdout, "cleaned %s\n", path.Base(podid))
	return
}

func cleanall() {
	pods, err := ioutil.ReadDir("media/")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		status = 1
		return
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
			status = 1
			die()
		}
	}
	_, err = os.Stat("media/" + path.Base(podid))
	if os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "media/%v doesn't exist, creating dir\n", path.Base(podid))
		err = os.Mkdir("media/"+path.Base(podid), 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			status = 1
			die()
		}
	}
	return
}

func main() {
	defer die()

	err := os.Chdir(os.Getenv("HOME") + "/pod/")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}

	if len(os.Args) < 2 {
		usage()
	}

	switch os.Args[1] {
	case "list":
		list()
	case "info":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "not enough arguments!\n")
			status = 1
			return
		}
		podid := os.Args[2]
		podInfo("rss/" + podid)
	case "fetch":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "not enough arguments!\n")
			status = 1
			return
		}
		for _, podid := range os.Args[2:] {
			err := fetchPodcast("rss/" + podid)
			if err == nil {
				fetchEpisode("rss/" + podid)
			}
		}
		return
	case "pull":
		pull()
	case "clean":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "not enough arguments!\n")
			status = 1
			return
		}
		if os.Args[2] == "all" {
			cleanall()
			return
		}
		clean("rss/" + os.Args[2])
		return
	case "refresh":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "not enough arguments!\n")
			status = 1
			return
		}
		for _, podid := range os.Args[2:] {
			fetchPodcast("rss/" + podid)
		}
	case "episode":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "not enough arguments!\n")
			status = 1
			return
		}
		podid := os.Args[2]
		podEpisode("rss/" + podid)
	case "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command\n")
		status = 1
		usage()
	}
	return
}
