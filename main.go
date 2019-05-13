package main

import (
	// "bytes"
	"fmt"
	// "github.com/emicklei/dot"
	"golang.org/x/net/html"
    "log"
	"net/http"
	// "strings"
)

func main() {
	h := http.NewServeMux()
    h.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "<html><body><a href=\"http://localhost:8080/test/another-page\">my link</a></body></html>")
    })
    h.HandleFunc("/test/another-page", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "<html><a href=\"http://localhost:8080/test\">my link</a></html>")
    })
    go http.ListenAndServe(":8080", h)

	startWebcrawler("http://localhost:8080/test/")

}

func startWebcrawler(domain string) string {
	// seenBefore := make(map[string]struct{})
	// seenBefore[domain] = struct{}{}

	// g := dot.NewGraph(dot.Directed)
	// startNode := g.Node(domain)

	// crawlFrom(domain, seenBefore, startNode, domain, g)

	// return g.String()

    startLink := make(chan string)
    foundLinks := make(chan string)

    log.Println("Starting linkGetter worker")
    go linkGetter(startLink, foundLinks)
    log.Println("linkGetter worker started")

    log.Println("pushing link to startLink channel")
    startLink <- domain

    for foundLink := range(foundLinks) {
        log.Printf(foundLink)
    }
    return ""
}

// func crawlFrom(domain string, seenBefore map[string]struct{}, parentNode dot.Node, currLink string, g *dot.Graph) {
// 	foundHyperlinks := make(chan string)
// 	go exploreForLinks(currLink, foundHyperlinks)
// 	resolvedUrlsInDomain := make(chan string)
// 	go filterExternalOrResolve(domain)(foundHyperlinks, resolvedUrlsInDomain)

// 	for resolvedUrl := range resolvedUrlsInDomain {

// 		currNode := g.Node(resolvedUrl)
// 		g.Edge(parentNode, currNode)

// 		if _, ok := seenBefore[resolvedUrl]; !ok { // if no seen before <-> not explored before
// 			seenBefore[resolvedUrl] = struct{}{} // mark it as seen
// 			crawlFrom(domain, seenBefore, currNode, resolvedUrl, g)
// 		}
// 	}
// }

func linkGetter(in chan string, out chan string) {
	for link := range in {
        log.Printf("link received %s", link)
		resp, err := http.Get(link) // GET
        log.Printf("have GOT from %s", link)
		if err == nil {
			defer resp.Body.Close()
			tokenizer := html.NewTokenizer(resp.Body)
			for {
				tokenType := tokenizer.Next()
                if (tokenType == html.ErrorToken) {
                    return
                }
                log.Println(tokenType)
				if tokenType == html.StartTagToken { // for each <a> tag
                    log.Println("Start tag token found")
                    token := tokenizer.Token()
					if token.Data == "a" {
                        log.Println("A tag found")
						for i := range token.Attr {// find the href attribute
                            log.Printf("Available key: %s", token.Attr[i].Key)
							if token.Attr[i].Key == "href" {
								go func() {
                                    out <- token.Attr[i].Val // and push to out channel
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

}

// func domainPrefixer(domain string)(in chan string, out chan string) {
// 	return func(in chan string, out chan string) {
// 		for url := range in {
// 			if strings.HasPrefix(url, "/") {
// 				out <- domain + url
// 			} else {
// 				out <- url
// 			}
// 		}
// 		close(out)
// 	}
// }

// func domainFilterer(domain string)(in chan string, out chan string) {
// 	return func(in chan string, out chan string) {
// 		for url := range in {
// 			if strings.HasPrefix(url, domain) {
// 				out <- url
// 			} else {
// 				// send another signal somehow
// 			}
// 		}
// 		close(out)
// 	}
// }

// func linkTidier(in chan string, out chan string) {
// 	for url := range in {
// 		out <- strings.Split(url, "#")[0] // remove fragment identifier
// 	}
// 	close(out)
// }

// func seenBeforeFilterer(in chan string, out chan string) {
// 	// keep a map of seen urls
// 	seenBefore := make(map[string]struct{})
// }

// func graphBuilder(in chan string, out chan string) {

// }

// type possibleLink struct {
// 	parentNode dot.Node
// 	String possibleUrl
// } 

// type goodLink struct {
// 	parentNode dot.Node
// 	goodLink dot.Node
// }


