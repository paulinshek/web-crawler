package main


import (
    "fmt"
    "log"
    "net/http"
	"golang.org/x/net/html"
	"bytes"
    "io/ioutil"
    "strings"
)

func everythingHandler(w http.ResponseWriter, r *http.Request) {
    // fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
    domain := "https://monzo.com"
    domainResolver := filterOrResolve(domain)

    foundHyperlinks := make(chan string)
    go getAndPushHyperlinksToChannel(domain, foundHyperlinks)

    resolvedUrlsInDomain := make(chan string)
    go func() {
        for foundHyperlink := range foundHyperlinks {
            domainResolver(foundHyperlink, resolvedUrlsInDomain)
        }
    }()

    go func() {
        for resolvedUrl := range resolvedUrlsInDomain {
            fmt.Println(resolvedUrl)
        }
    }()

}

func main() {
    http.HandleFunc("/", everythingHandler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func getAndPushHyperlinksToChannel(url string, c chan string) {
	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err == nil {
		bodyAsByteArray, _ := ioutil.ReadAll(resp.Body)
		doc, _ := html.Parse(bytes.NewReader(bodyAsByteArray))

		pushHyperlinksToChannelRecursively(doc, c)

	} else{
		fmt.Println("ERROR: %s", err)
	}

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

func filterOrResolve(domain string) func(url string, c chan string) {
    return func(url string, c chan string) {
        if strings.HasPrefix(url, "/") {
            c <- domain + url
        } else if strings.HasPrefix(url, domain) {
            c <- url
        }
    }
}