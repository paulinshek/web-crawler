package main

import (
	"fmt"
	"github.com/emicklei/dot"
	"golang.org/x/net/html"
	"log"
	"net/http"
	"strings"
)

func main() {
	h := http.NewServeMux()
	h.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<html><a href=\"/test/another-page\">my link</a><a href=\"http://otherdomain.com\">exclude me</a></html>")
	})
	h.HandleFunc("/test/another-page", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<a href=\"http://localhost:8080/test\">my link</a>")
	})
	go http.ListenAndServe(":8080", h)

	startWebcrawler("http://localhost:8080/test")

}

func startWebcrawler(domain string) string {
	cLinkGetter := make(chan string)
	cDomainPrefixer := make(chan parentChildPair)
	cDomainFilterer := make(chan parentChildPair)
	cGraphBuilder := make(chan parentChildPair)
	cResultGraph := make(chan string)

	go linkGetter(cLinkGetter, cDomainPrefixer)
	go domainPrefixer("http://localhost:8080", cDomainPrefixer, cDomainFilterer)
	go domainFilterer("http://localhost:8080", cDomainFilterer, cGraphBuilder)
	go graphBuilder(cGraphBuilder, cLinkGetter, cResultGraph)

	log.Println("pushing link to graphbuilder to make the first node")
	cGraphBuilder <- parentChildPair{childLink: domain}

	resultGraph := <-cResultGraph
	log.Printf("FINAL RESULT: %s", resultGraph)
	return resultGraph
}

type parentChildPair struct {
	parentLink string
	childLink string
}

func linkGetter(in chan string, out chan parentChildPair) {
	for link := range in {
		log.Printf("link received %s", link)
		resp, err := http.Get(link) // GET
		log.Printf("have GOT from %s", link)
		if err == nil {
			defer resp.Body.Close()
			tokenizer := html.NewTokenizer(resp.Body)
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
								go func() {
									out <- parentChildPair{parentLink: link, childLink: token.Attr[i].Val}
								}()
							}
						}
					}
				}
			}
		} else {
			log.Println("ERROR: %s", err)
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
			// send another signal somehow
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

func graphBuilder(in chan parentChildPair, outBackToLinkGetter chan string, finalOutput chan string) {
	g := dot.NewGraph(dot.Directed)
	seenBefore := make(map[string]dot.Node)

	// whilst loop detection has not been implemented
	// end artificially
	maxItems := 3
	count := 0
	for parentChild := range in {
		count++
		log.Printf("itercepted parentChild: %s", parentChild)

		// first sort out node creation
		// if child seen before then get the already existing node instead
		childNode, childSeenBefore := seenBefore[parentChild.childLink]
		if childSeenBefore {
			// TODO don't need to go back round ie. merge channelInterceptor and graph builder
		} else {
			childNode = g.Node(parentChild.childLink)
			seenBefore[parentChild.childLink] = childNode
			outBackToLinkGetter <- parentChild.childLink
		}

		// now add an edge if needed
		if len(parentChild.parentLink) > 0 {
			parentNode, _ := seenBefore[parentChild.parentLink]
			g.Edge(parentNode, childNode)
		}

		// TODO somehow need to work out when everything's been explored
		if count == maxItems {
			break
		}
	}
	close(outBackToLinkGetter)
	
	finalOutput <- g.String()
	close(finalOutput)
}
