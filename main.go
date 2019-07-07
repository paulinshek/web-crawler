package main

import (
	"fmt"
	"github.com/emicklei/dot"
	"golang.org/x/net/html"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
)

func main() {
	baseUrlString := os.Args[1]

	fmt.Println(startWebcrawler(baseUrlString).String())
}

func startWebcrawler(start string) dot.Graph {
	cStartURL := make(chan *url.URL)

	cLinkGetter := make(chan *url.URL, 10000) // max number of links on one page ^ 2
	cExploredURL := make(chan ExploredURL, 1)

	cDomainFilterer := make(chan parentChildPair)
	cParentLinkWithFilteredChild := make(chan *url.URL)
	cGraphBuilder := make(chan parentChildPair)
	cResultGraph := make(chan dot.Graph)

	startUrl, err := url.Parse(start)

	var wg sync.WaitGroup
	const numLinkGetters = 20
	wg.Add(numLinkGetters)
	for i := 0; i < numLinkGetters; i++ {
		go func() {
			linkGetter(startUrl, cLinkGetter, cDomainFilterer, cExploredURL)
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(cExploredURL)
	}()

	if err != nil {
		log.Printf("ERROR: Error parsing start url string: %s", start)
		log.Printf("ERROR: %#v", err)
		log.Printf("ERROR: Returning empty graph")
		return *dot.NewGraph(dot.Directed)
	}
	domain := startUrl.Hostname()

	go domainFilterer(domain, cDomainFilterer, cGraphBuilder, cParentLinkWithFilteredChild)
	go graphBuilder(cStartURL, cGraphBuilder, cParentLinkWithFilteredChild, cExploredURL, cLinkGetter, cResultGraph)

	log.Println("pushing link to graphbuilder to make the first node")
	cStartURL <- startUrl

	resultGraph := <-cResultGraph
	return resultGraph
}

type parentChildPair struct {
	parentLink *url.URL
	childLink  *url.URL
}

func linkGetter(baseUrl *url.URL, in <-chan *url.URL, out chan<- parentChildPair, cExploredURL chan<- ExploredURL) {
	for link := range in {
		log.Printf("link received %#v", link)
		resp, err := http.Get(link.String()) // GET
		log.Printf("have GOT from %s", link.String())
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
								childLinkUrl, _ := url.Parse(token.Attr[i].Val)
								out <- parentChildPair{parentLink: link, childLink: baseUrl.ResolveReference(childLinkUrl)}
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

func domainFilterer(base string, in <-chan parentChildPair, goodOut chan<- parentChildPair, badOut chan<- *url.URL) {
	for parentChild := range in {
		if parentChild.childLink.Hostname() == base {
			goodOut <- parentChild
		} else {
			log.Printf("INFO: bad link %#v", parentChild)
			log.Printf("INFO: childUrl.Hostname %s and base %s", parentChild.childLink.Hostname(), base)
			badOut <- parentChild.parentLink
		}
	}
	close(goodOut)
	close(badOut)
}



// ExploredURL signals when an link has been GOT and all its children have been sent
type ExploredURL struct {
	url                   *url.URL
	numberOfChildrenCount int
}

func graphBuilder(
	cStartURL chan *url.URL,
	cParentChildPair <-chan parentChildPair,
	cParentWithFilteredChild <-chan *url.URL,
	cExploredURLs <-chan ExploredURL,
	outBackToLinkGetter chan<- *url.URL,
	finalOutput chan dot.Graph) {

	// stores the graph that we will output at the end
	g := dot.NewGraph(dot.Directed)
	// stops us from getting into infinite loops
	seenBefore := make(map[string]dot.Node)
	// keeps track of the number of children seen vs the number of children we expect to see 
	// per parent, and then tells us when we are done
	state := NewStateTracker()

	startURL := <-cStartURL
	log.Printf("Start url %#v received and creating new node for it", startURL)
	startURLPath := startURL.Path
	rootNode := g.Node(startURLPath)
	seenBefore[startURLPath] = rootNode
	state.InitialiseRoot(startURLPath)
	outBackToLinkGetter <- startURL
	close(cStartURL)

	var allExplored = false
	for !allExplored {
		select {
		case parentChild := <-cParentChildPair:
			// link received
			log.Printf("received parentChild: %#v", parentChild)

			parentLinkPath := parentChild.parentLink.Path
			childLinkPath := parentChild.childLink.Path

			state.ChildOfParent(parentLinkPath)

			// sort out node creation
			// if child seen before then get the already existing node instead
			childNode, childSeenBefore := seenBefore[childLinkPath]
			if !childSeenBefore {
				log.Println("Child not seen before, so creating new node for it")
				childNode = g.Node(childLinkPath)
				seenBefore[childLinkPath] = childNode
				state.NewChildFound(childLinkPath)
				outBackToLinkGetter <- parentChild.childLink
			}

			// now add an edge
			parentNode, err := seenBefore[parentLinkPath]
			if err {
				log.Println("ERROR: Error getting parent node")
			}
			log.Println("Adding edge from parent node to child node")
			g.Edge(parentNode, childNode)

		case parentLink := <-cParentWithFilteredChild:
			// link received
			log.Printf("received parent with filtered child: %#v", parentLink)
			state.ChildOfParent(parentLink.Path)

		case exploredURL := <-cExploredURLs:
			log.Printf("received exploredURL: %#v", exploredURL)
			state.ParentExplored(exploredURL.url.Path, exploredURL.numberOfChildrenCount)
		}

		allExplored = state.IsAllExplored()
	}
	close(outBackToLinkGetter)

	finalOutput <- *g
	close(finalOutput)
}
