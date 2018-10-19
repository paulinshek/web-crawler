package main

import (
	"bytes"
	"fmt"
	"golang.org/x/net/html"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func startTestServer() {
	h := http.NewServeMux()
	h.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<a href=\"/another-page\">my link</a>")
	})
	h.HandleFunc("/another-page", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "yayyy")
	})
	log.Fatal(http.ListenAndServe(":8080", h))
}

func startWebcrawlerServer() {
	h := http.NewServeMux()
	h.HandleFunc("/", startCrawling)
	log.Fatal(http.ListenAndServe(":8000", h))
}

func startCrawling(w http.ResponseWriter, r *http.Request) {
	domain := "http://localhost:8080/test"
	seenBefore := make(map[string]struct{})
	seenBefore[domain] = struct{}{}
	crawlFrom(domain, seenBefore, domain, w)
}

func crawlFrom(domain string, seenBefore map[string]struct{}, currLink string, w http.ResponseWriter) {

	foundHyperlinks := make(chan string)
	go exploreForLinks(currLink, foundHyperlinks)
	resolvedUrlsInDomain := make(chan string)
	go filterExternalOrResolve(domain)(foundHyperlinks, resolvedUrlsInDomain)

	for resolvedUrl := range resolvedUrlsInDomain {
		if _, ok := seenBefore[resolvedUrl]; !ok { // if no seen before <-> not explored before
			seenBefore[resolvedUrl] = struct{}{} // mark it as seen

			crawlFrom(domain, seenBefore, resolvedUrl, w)
		}
	}
	fmt.Println(w, currLink)
}

func main() {
	go startTestServer()

	go startWebcrawlerServer()

	resp, err := http.Get("http://localhost:8000/")
	if err == nil {
		defer resp.Body.Close()
		bodyAsByteArray, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(bodyAsByteArray[:]))
	}
}

func exploreForLinks(currUrl string, out chan string) {
	resp, err := http.Get(currUrl)
	if err == nil {
		defer resp.Body.Close()
		bodyAsByteArray, _ := ioutil.ReadAll(resp.Body)
		doc, _ := html.Parse(bytes.NewReader(bodyAsByteArray))
		pushHyperlinksToChannelRecursively(doc, out)

	} else {
		fmt.Println("ERROR: %s", err)
	}
	close(out)
}

func pushHyperlinksToChannelRecursively(n *html.Node, rawHyperlinkReceiver chan string) {
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, a := range n.Attr {
			if a.Key == "href" {
				rawHyperlinkReceiver <- a.Val
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		pushHyperlinksToChannelRecursively(c, rawHyperlinkReceiver)
	}
}

func filterExternalOrResolve(domain string) func(in chan string, out chan string) {
	return func(in chan string, out chan string) {
		for url := range in {
			if strings.HasPrefix(url, "/") {
				out <- removeFragmentIdentifier(domain + url)
			} else if strings.HasPrefix(url, domain) {
				out <- removeFragmentIdentifier(url)
			}
		}
		close(out)
	}
}

func removeFragmentIdentifier(url string) string {
	return strings.Split(url, "#")[0]
}

// BUG printed urls are not completely unique
// TODO parent link should also be pushed to channel so that we can create a graph from this Data
// TODO actually return something
