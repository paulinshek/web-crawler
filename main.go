package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/emicklei/dot"
	"golang.org/x/net/html"
)

func main() {
	h := http.NewServeMux()
	h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<html><a href=\"/another-page#56765\">my link</a><a href=\"http://otherdomain.com\">exclude me</a></html>")
	})
	h.HandleFunc("/another-page", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<a href=\"http://localhost:8080/\">my link back</a><a href=\"http://localhost:8080/another-page\">my link to myself</a>")
	})
	go http.ListenAndServe(":8080", h)

	fmt.Println(startWebcrawler("http://localhost:8080/", "http://localhost:8080").String())
	// startWebcrawler("http://monzo.com/", "http://monzo.com")
}

func startWebcrawler(start string, domain string) dot.Graph {
	cStartURL := make(chan string)

	cLinkGetter := make(chan string)
	cExploredURL := make(chan ExploredURL)

	cDomainPrefixer := make(chan parentChildPair)
	cDomainFilterer := make(chan parentChildPair)
	cLinkTidier := make(chan parentChildPair)
	cGraphBuilder := make(chan parentChildPair)
	cResultGraph := make(chan dot.Graph)

	go linkGetter(cLinkGetter, cDomainPrefixer, cExploredURL)
	go domainPrefixer(domain, cDomainPrefixer, cDomainFilterer)
	go domainFilterer(domain, cDomainFilterer, cLinkTidier)
	go linkTidier(cLinkTidier, cGraphBuilder)
	go graphBuilder(cStartURL, cGraphBuilder, cExploredURL, cLinkGetter, cResultGraph)

	log.Println("pushing link to graphbuilder to make the first node")
	cStartURL <- start

	resultGraph := <-cResultGraph
	return resultGraph
}

type parentChildPair struct {
	parentLink string
	childLink  string
}

func linkGetter(in chan string, out chan parentChildPair, cExploredURL chan ExploredURL) {
	for link := range in {
		log.Printf("link received %s", link)
		resp, err := http.Get(link) // GET
		log.Printf("have GOT from %s", link)
		if err == nil {
			defer resp.Body.Close()
			tokenizer := html.NewTokenizer(resp.Body)
			childrenCount := 0
			for {
				tokenType := tokenizer.Next()
				if tokenType == html.ErrorToken {
					break
				}
				log.Printf("token type: %s", tokenType)
				if tokenType == html.StartTagToken {
					token := tokenizer.Token()
					log.Printf("token: %s", token)
					log.Printf("token data: %s", token.Data)
					if token.Data == "a" { // for each <a> tag
						for i := range token.Attr { // find the href attribute
							log.Printf("Available key: %s", token.Attr[i].Key)
							if token.Attr[i].Key == "href" {
								childrenCount++
								go func() {
									out <- parentChildPair{parentLink: link, childLink: token.Attr[i].Val}
								}()
							}
						}
					}
				}
			}
			cExploredURL <- ExploredURL{link, childrenCount}
		} else {
			log.Printf("ERROR: %s", err)
		}
	}
	close(out)
}

func domainPrefixer(domain string, in chan parentChildPair, out chan parentChildPair) {
	for parentChild := range in {
		if strings.HasPrefix(parentChild.childLink, "/") {
			parentChild.childLink = domain + parentChild.childLink
		}
		out <- parentChild
	}
	close(out)
}

func domainFilterer(domain string, in chan parentChildPair, out chan parentChildPair) {
	for parentChild := range in {
		if strings.HasPrefix(parentChild.childLink, domain) {
			out <- parentChild
		} else {
			out <- parentChildPair{parentLink: parentChild.parentLink, childLink: ""}
		}
	}
	close(out)
}

func linkTidier(in chan parentChildPair, out chan parentChildPair) {
	for parentChild := range in {
		withoutFragmentIdentifier := strings.Split(parentChild.childLink, "#")[0]
		parentChild.childLink = withoutFragmentIdentifier
		out <- parentChild
	}
	close(out)
}

// ChildrenCount keeps track of a signal URL: the number of children that have been found
// so far (from GET-ing their parent) vs the number of children that have been received
// down the pipeline
// when these two numbers are equal we know that we are done for this parent
type ChildrenCount struct {
	numberOfFoundChildren    int
	numberOfReceivedChildren int
}

// ExploredURL signals when an link has been GOT and all its children have been sent
type ExploredURL struct {
	url                   string
	numberOfChildrenCount int
}

func graphBuilder(
	cStartURL chan string,
	cParentChildPair chan parentChildPair,
	cExploredURLs chan ExploredURL,
	outBackToLinkGetter chan string,
	finalOutput chan dot.Graph) {

	g := dot.NewGraph(dot.Directed)
	seenBefore := make(map[string]dot.Node)
	childrenCountMap := make(map[string]ChildrenCount)

	startURL := <-cStartURL
	rootNode := g.Node(startURL)
	seenBefore[startURL] = rootNode
	childrenCountMap[startURL] = ChildrenCount{numberOfFoundChildren: -1, numberOfReceivedChildren: 0}
	outBackToLinkGetter <- startURL
	close(cStartURL)

	var allExplored = false
	for !allExplored {
		select {
		case parentChild := <-cParentChildPair:
			// link received
			log.Printf("received parentChild: %#v", parentChild)

			// update the counts
			oldCounts := childrenCountMap[parentChild.parentLink]
			newCounts := ChildrenCount{
				numberOfFoundChildren:    oldCounts.numberOfFoundChildren,
				numberOfReceivedChildren: oldCounts.numberOfReceivedChildren + 1}
			childrenCountMap[parentChild.parentLink] = newCounts

			// sort out node creation
			// if child seen before then get the already existing node instead
			childNode, childSeenBefore := seenBefore[parentChild.childLink]
			if !childSeenBefore {
				childNode = g.Node(parentChild.childLink)
				seenBefore[parentChild.childLink] = childNode
				childrenCountMap[parentChild.childLink] = ChildrenCount{numberOfFoundChildren: -1, numberOfReceivedChildren: 0}
				outBackToLinkGetter <- parentChild.childLink
			}
			// now add an edge
			parentNode, _ := seenBefore[parentChild.parentLink]
			g.Edge(parentNode, childNode)
		case exploredURL := <-cExploredURLs:
			log.Printf("recevied exploredURL: %#v", exploredURL)
			// mark as explored and update total count
			oldChildrenCount := childrenCountMap[exploredURL.url]
			newChildrenCount := ChildrenCount{
				numberOfReceivedChildren: oldChildrenCount.numberOfReceivedChildren,
				numberOfFoundChildren:    exploredURL.numberOfChildrenCount}

			childrenCountMap[exploredURL.url] = newChildrenCount

			// check if everything has been explored
			allExplored = true
			for _, value := range childrenCountMap {
				allExplored = allExplored &&
					value.numberOfReceivedChildren == value.numberOfFoundChildren
			}
			log.Printf("childrenCountMap %#v", childrenCountMap)
		}
	}
	close(outBackToLinkGetter)

	finalOutput <- *g
	close(finalOutput)
}
