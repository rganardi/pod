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
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	humanize "github.com/dustin/go-humanize"
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
	status                     int    = 0
	msg                        io.Writer
	log                        io.Writer
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
	fetch PODCAST EPSID	get episode EPSID of PODCAST
	pull			get the latest episode of all podcasts
	clean PODCAST		remove media files for PODCAST
	episode PODCAST		see all episodes of PODCAST
	help			display this help
`)
	return
}

func fetch(url, fileName string) {
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

	filesize, _ := strconv.Atoi(response.Header.Get("Content-Length"))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err = io.Copy(output, response.Body)
		output.Sync()
	}()

	var cursize int64 = 0
	for {
		stat, err := output.Stat()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		dlspeed := stat.Size() - cursize
		if int(stat.Size()) < filesize {
			fmt.Fprintf(msg, "(%2v%% %6v/s) %6v\r", (int(stat.Size()) * 100 / filesize), humanize.Bytes(uint64(dlspeed)), humanize.Bytes(uint64(filesize)))
		} else {
			break
		}
		cursize = stat.Size()
		time.Sleep(time.Second)
	}

	wg.Wait()

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
		fmt.Fprintf(log, "%v\n", file.Name())
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
	d := xml.NewDecoder(bytes.NewReader(file))
	err = d.Decode(&q)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while decoding %s\n", filename)
		fmt.Fprintf(os.Stderr, "%v\n", err)
		status = 1
		return
	}
	c := q.Podcast

	fmt.Fprintf(log, "title\t\t%v\n", c.Title)

	for _, feedurl := range c.NewFeedUrl {
		if feedurl.Rel == "self" {
			fmt.Fprintf(log, "url\t\t%v\n", feedurl.Link)
		}
	}
	lastEpisode := c.EpisodeList[0]
	fmt.Fprintf(log, "last episode\t%v\n\t\t%v\n", lastEpisode.PubDate, lastEpisode.Title)
	fmt.Fprintf(log, "desc\t\t%v\n", lastEpisode.Desc)
	return
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
	commandToRun.Stdout = log
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

	for epsid, episode := range c.EpisodeList {
		fmt.Fprintf(pagerStdin, "id\t\t%v\n", epsid)
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
}

func fetchEpisode(podid string, epsid int) {
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

	url := c.EpisodeList[epsid].Enclosure.Link
	filename := "media/" + podname + "/" + path.Base(url)

	//check if the file to download already exists
	_, err = os.Stat(filename)
	if err == nil {
		//fmt.Fprintf(log, "media already downloaded\n")
		return
	}
	fmt.Fprintf(msg, "%-20s %-30s %-30s\r", podname, c.EpisodeList[epsid].Title, "fetching")
	fetch(url, filename)
	fmt.Fprintf(log, "%-20s %-30s\n", podname, c.EpisodeList[epsid].Title)
	return
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
		fmt.Fprintf(msg, "%-50s %-30s\r", file.Name(), "fetching")
		err = fetchPodcast("rss/" + file.Name())
		if err != nil {
			//don't download the episode
			continue
		}
		fetchEpisode("rss/"+file.Name(), 0)
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
		fmt.Fprintf(msg, "%s %-50s\r", "cleaning", pod.Name())
		clean("rss/" + pod.Name())
	}

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

	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		msg = os.Stdout
		log = os.Stdout
	} else {
		msg, err = os.Open("/dev/null")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		log = os.Stdout
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
		podid := os.Args[2]
		var epsid int
		if len(os.Args) < 4 {
			epsid = 0
		} else {
			epsid, err = strconv.Atoi(os.Args[3])
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
				status = 1
				return
			}
		}
		err = fetchPodcast("rss/" + podid)
		if err == nil {
			fetchEpisode("rss/"+podid, epsid)
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
