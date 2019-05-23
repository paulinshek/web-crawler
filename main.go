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
	h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<html><a href=\"/another-page#56765\">my link</a><a href=\"http://otherdomain.com\">exclude me</a></html>")
	})
	h.HandleFunc("/another-page", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<a href=\"http://localhost:8080/\">my link back</a><a href=\"http://localhost:8080/another-page\">my link to myself</a><a href=\"http://localhost:8080/no-links\">link to a page with no links</a>")
	})
	h.HandleFunc("/no-links", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "")
	})
	go http.ListenAndServe(":8080", h)

	startWebcrawler("http://localhost:8080/", "http://localhost:8080")
	// startWebcrawler("http://monzo.com/", "http://monzo.com")
}

func startWebcrawler(start string, domain string) string {
	cLinkGetter := make(chan string)
	cDomainPrefixer := make(chan parentChildPair)
	cDomainFilterer := make(chan parentChildPair)
	cLinkTidier := make(chan parentChildPair)
	cGraphBuilder := make(chan parentChildPair)
	cResultGraph := make(chan string)

	go linkGetter(cLinkGetter, cDomainPrefixer)
	go domainPrefixer(domain, cDomainPrefixer, cDomainFilterer)
	go domainFilterer(domain, cDomainFilterer, cLinkTidier)
	go linkTidier(cLinkTidier, cGraphBuilder)
	go graphBuilder(cGraphBuilder, cLinkGetter, cResultGraph)

	log.Println("pushing link to graphbuilder to make the first node")
	cGraphBuilder <- parentChildPair{childLink: start}

	resultGraph := <-cResultGraph
	log.Printf("FINAL RESULT: %s", resultGraph)
	return resultGraph
}

type parentChildPair struct {
	parentLink                 string
	childLink                  string
	numberOfChildrenFoundSoFar int
}

func isHtmlContent(contentTypeList []string) bool {
	for i := range contentTypeList {
		log.Printf("contentType: %s", contentTypeList[i])
		if strings.Contains(contentTypeList[i], "text/html") {
			return true
		}
	}
	return false
}

func linkGetter(in chan string, out chan parentChildPair) {
	for link := range in {
		log.Printf("link received %s", link)
		resp, err := http.Get(link) // GET
		log.Printf("have GOT from %s", link)

		isHtml := isHtmlContent(resp.Header["Content-Type"])
		log.Printf("isHtml: %s", isHtml)
		if err == nil && isHtml {
			childrenCount := 0
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
								childrenCount++
								go func() {
									out <- parentChildPair{parentLink: link, childLink: token.Attr[i].Val, numberOfChildrenFoundSoFar: childrenCount}
								}()
							}
						}
					}
				}
			}
		}
		if err != nil {
			log.Println("ERROR: %s", err)
		}
		if !isHtml {
			log.Println("%s is not html", link)
			go func() {
				out <- parentChildPair{parentLink: link, numberOfChildrenFoundSoFar: 0}
			}()
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
			out <- parentChildPair{parentLink: parentChild.parentLink, childLink: "", numberOfChildrenFoundSoFar: parentChild.numberOfChildrenFoundSoFar}
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

type childrenCount struct {
	numberOfFoundChildren    int
	numberOfExploredChildren int
}

func graphBuilder(in chan parentChildPair, outBackToLinkGetter chan string, finalOutput chan string) {
	g := dot.NewGraph(dot.Directed)
	seenBefore := make(map[string]dot.Node)
	childrenCountMap := make(map[string]childrenCount)

	for parentChild := range in {
		log.Printf("itercepted parentChild: %s", parentChild)

		// update the counts
		if len(parentChild.parentLink) > 0 && len(parentChild.childLink) > 0 {
			oldCounts := childrenCountMap[parentChild.parentLink]
			var numberOfFoundChildren int = oldCounts.numberOfFoundChildren
			if parentChild.numberOfChildrenFoundSoFar > numberOfFoundChildren {
				numberOfFoundChildren = parentChild.numberOfChildrenFoundSoFar
			}
			newCounts := childrenCount{numberOfFoundChildren: numberOfFoundChildren, numberOfExploredChildren: oldCounts.numberOfExploredChildren + 1}
			childrenCountMap[parentChild.parentLink] = newCounts
			
		} else if len(parentChild.parentLink) > 0 && len(parentChild.childLink) == 0 {
			oldCounts := childrenCountMap[parentChild.parentLink]
			var numberOfFoundChildren int = oldCounts.numberOfFoundChildren
			if parentChild.numberOfChildrenFoundSoFar > numberOfFoundChildren {
				numberOfFoundChildren = parentChild.numberOfChildrenFoundSoFar
			}
			newCounts := childrenCount{numberOfFoundChildren: numberOfFoundChildren, numberOfExploredChildren: oldCounts.numberOfExploredChildren}
			childrenCountMap[parentChild.parentLink] = newCounts
		}
		// first sort out node creation
		// if child seen before then get the already existing node instead
		childNode, childSeenBefore := seenBefore[parentChild.childLink]
		if childSeenBefore || len(parentChild.childLink) == 0 || parentChild.parentLink == parentChild.childLink {
			// don't need to go back round
		} else if (len(parentChild.childLink) > 0) {
			childNode = g.Node(parentChild.childLink)
			seenBefore[parentChild.childLink] = childNode
			childrenCountMap[parentChild.childLink] = childrenCount{numberOfFoundChildren: -1, numberOfExploredChildren: 0}
			outBackToLinkGetter <- parentChild.childLink

			// now add an edge if needed
			if len(parentChild.parentLink) > 0 {
				parentNode, _ := seenBefore[parentChild.parentLink]
				g.Edge(parentNode, childNode)
			}
		}

		// check if everything has been explored
		var allExplored bool = true
		for _, value := range childrenCountMap {
			allExplored = allExplored &&
				value.numberOfExploredChildren == value.numberOfFoundChildren &&
				value.numberOfFoundChildren >= 0
		}
		log.Printf("childrenCountMap %s", childrenCountMap)
		if allExplored {
			break
		}
	}
	close(outBackToLinkGetter)

	finalOutput <- g.String()
	close(finalOutput)
}
