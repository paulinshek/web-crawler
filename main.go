package main

import (
	"fmt"
	"github.com/emicklei/dot"
	"golang.org/x/net/html"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

func main() {
	root := os.Args[1]
	fmt.Println(startWebcrawler(root, root).String())
}

func startWebcrawler(start string, domain string) dot.Graph {
	cStartURL := make(chan string)

	cLinkGetter := make(chan string, 10000) // max number of links on one page ^ 2
	cExploredURL := make(chan ExploredURL, 1)

	cDomainPrefixer := make(chan parentChildPair)
	cDomainFilterer := make(chan parentChildPair)
	cLinkTidier := make(chan parentChildPair)
	cParentLinkWithFilteredChild := make(chan string)
	cGraphBuilder := make(chan parentChildPair)
	cResultGraph := make(chan dot.Graph)

	var wg sync.WaitGroup
	const numLinkGetters = 20
	wg.Add(numLinkGetters)
	for i := 0; i < numLinkGetters; i++ {
		go func() {
			linkGetter(cLinkGetter, cDomainPrefixer, cExploredURL)
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(cExploredURL)
	}()

	go domainPrefixer(domain, cDomainPrefixer, cDomainFilterer)
	go domainFilterer(domain, cDomainFilterer, cLinkTidier, cParentLinkWithFilteredChild)
	go linkTidier(cLinkTidier, cGraphBuilder)
	go graphBuilder(cStartURL, cGraphBuilder, cParentLinkWithFilteredChild, cExploredURL, cLinkGetter, cResultGraph)

	log.Println("pushing link to graphbuilder to make the first node")
	cStartURL <- start

	resultGraph := <-cResultGraph
	return resultGraph
}

type parentChildPair struct {
	parentLink string
	childLink  string
}

func linkGetter(in <-chan string, out chan<- parentChildPair, cExploredURL chan<- ExploredURL) {
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
								out <- parentChildPair{parentLink: link, childLink: token.Attr[i].Val}
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
}

func domainPrefixer(domain string, in <-chan parentChildPair, out chan<- parentChildPair) {
	for parentChild := range in {
		if strings.HasPrefix(parentChild.childLink, "/") {
			parentChild.childLink = domain + parentChild.childLink
		}
		out <- parentChild
	}
	close(out)
}

func domainFilterer(domain string, in <-chan parentChildPair, goodOut chan<- parentChildPair, badOut chan<- string) {
	for parentChild := range in {
		if strings.HasPrefix(parentChild.childLink, domain) {
			goodOut <- parentChild
		} else {
			badOut <- parentChild.parentLink
		}
	}
	close(goodOut)
	close(badOut)
}

func linkTidier(in <-chan parentChildPair, out chan<- parentChildPair) {
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
	cParentChildPair <-chan parentChildPair,
	cParentWithFilteredChild <-chan string,
	cExploredURLs <-chan ExploredURL,
	outBackToLinkGetter chan<- string,
	finalOutput chan dot.Graph) {

	g := dot.NewGraph(dot.Directed)
	seenBefore := make(map[string]dot.Node)
	childrenCountMap := make(map[string]ChildrenCount)

	startURL := <-cStartURL
	log.Println("Start node received and creating new node for it")
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
				log.Println("Child not seen before, so creating new node for it")
				childNode = g.Node(parentChild.childLink)
				seenBefore[parentChild.childLink] = childNode
				childrenCountMap[parentChild.childLink] = ChildrenCount{numberOfFoundChildren: -1, numberOfReceivedChildren: 0}
				outBackToLinkGetter <- parentChild.childLink

			}
			// now add an edge
			parentNode, err := seenBefore[parentChild.parentLink]
			if err {
				log.Println("Error getting parent node")
			}
			log.Println("Adding edge from parent node to child node")
			g.Edge(parentNode, childNode)

			// check if everything has been explored
			allExplored = true
			for _, value := range childrenCountMap {
				allExplored = allExplored &&
					value.numberOfReceivedChildren == value.numberOfFoundChildren
			}
			log.Printf("childrenCountMap %#v", childrenCountMap)
		case parentLink := <-cParentWithFilteredChild:
			// link received
			log.Printf("received parent with filtered child: %#v", parentLink)

			// update the counts
			oldCounts := childrenCountMap[parentLink]
			newCounts := ChildrenCount{
				numberOfFoundChildren:    oldCounts.numberOfFoundChildren,
				numberOfReceivedChildren: oldCounts.numberOfReceivedChildren + 1}
			childrenCountMap[parentLink] = newCounts

			// check if everything has been explored
			allExplored = true
			for _, value := range childrenCountMap {
				allExplored = allExplored &&
					value.numberOfReceivedChildren == value.numberOfFoundChildren
			}
			log.Printf("childrenCountMap %#v", childrenCountMap)
		case exploredURL := <-cExploredURLs:
			log.Printf("received exploredURL: %#v", exploredURL)
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
