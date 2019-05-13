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
	cDomainPrefixer := make(chan string)
	cDomainFilterer := make(chan string)
	cFoundLinks := make(chan string)
	cGraphBuilder := make(chan string)
	cResultGraph := make(chan string)

	go linkGetter(cLinkGetter, cDomainPrefixer)
	go domainPrefixer("http://localhost:8080", cDomainPrefixer, cDomainFilterer)
	go domainFilterer("http://localhost:8080", cDomainFilterer, cFoundLinks)
	// feed back into first channel to create a cycle
	// and also feed to graphBuilder which collects results
	go channelInterceptor(cFoundLinks, cGraphBuilder, cLinkGetter) // also does some logging
	go graphBuilder(cGraphBuilder, cResultGraph)

	log.Println("pushing link to startLink channel")
	cLinkGetter <- domain

	resultGraph := <-cResultGraph
	log.Printf("FINAL RESULT: %s", resultGraph)
	return resultGraph
}

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
	close(out)
}

func domainPrefixer(domain string, in chan string, out chan string) {
	for url := range in {
		if strings.HasPrefix(url, "/") {
			out <- domain + url
		} else {
			out <- url
		}
	}
	close(out)
}

func domainFilterer(domain string, in chan string, out chan string) {
	for url := range in {
		if strings.HasPrefix(url, domain) {
			out <- url
		} else {
			// send another signal somehow
		}
	}
	close(out)
}

func linkTidier(in chan string, out chan string) {
	for url := range in {
		out <- strings.Split(url, "#")[0] // remove fragment identifier
	}
	close(out)
}

// func seenBeforeFilterer(in chan string, out chan string) {
// 	// keep a map of seen urls
// seenBefore := make(map[string]struct{})
// seenBefore[domain] = struct{}{}
// }

func graphBuilder(in chan string, out chan string) {
	g := dot.NewGraph(dot.Directed)
	for url := range in {
		g.Node(url)
	}
	out <- g.String()
	close(out)
}

func channelInterceptor(in chan string, out1 chan string, out2 chan string) {
	// whilst loop detection has not been implemented
	// end artificially
	maxItems := 2
	count := 0
	for item := range in {
		count++
		log.Printf("itercepted item: %s", item)
		go func() {
			out1 <- item
		}()
		go func() {
			out2 <- item
		}()
		if count == maxItems {
			break
		}
	}
	<-in // manually filter out seen link
	close(out1)
	close(out2)
}

// type possibleLink struct {
// 	parentNode dot.Node
// 	String possibleUrl
// }

// type goodLink struct {
// 	parentNode dot.Node
// 	goodLink dot.Node
// }
