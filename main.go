package main

import (
	"bytes"
	"fmt"
	"github.com/emicklei/dot"
	"golang.org/x/net/html"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

func main() {
	root := os.Args[1]
	fmt.Printf(startWebcrawler(root))

}

func startWebcrawler(domain string) string {
	seenBefore := make(map[string]struct{})
	seenBefore[domain] = struct{}{}

	g := dot.NewGraph(dot.Directed)
	startNode := g.Node(domain)

	crawlFrom(domain, seenBefore, startNode, domain, g)

	return g.String()
}

func crawlFrom(domain string, seenBefore map[string]struct{}, parentNode dot.Node, currLink string, g *dot.Graph) {
	foundHyperlinks := make(chan string)
	go exploreForLinks(currLink, foundHyperlinks)
	resolvedUrlsInDomain := make(chan string)
	go filterExternalOrResolve(domain)(foundHyperlinks, resolvedUrlsInDomain)

	for resolvedUrl := range resolvedUrlsInDomain {

		currNode := g.Node(resolvedUrl)
		g.Edge(parentNode, currNode)

		if _, ok := seenBefore[resolvedUrl]; !ok { // if no seen before <-> not explored before
			seenBefore[resolvedUrl] = struct{}{} // mark it as seen
			crawlFrom(domain, seenBefore, currNode, resolvedUrl, g)
		}
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
