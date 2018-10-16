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
    domain := "https://monzo.com"
    domainResolver := filterOrResolve(domain)

    unexploredUrls := make(chan string)
    go func() {
        unexploredUrls <- domain // start
    }()

    foundHyperlinks := make(chan string)
    go func() {
        for unexploredUrl := range unexploredUrls {
            getAndPushHyperlinksToChannel(unexploredUrl, foundHyperlinks)
        }
    }()

    resolvedUrlsInDomain := make(chan string)
    go func() {
        for foundHyperlink := range foundHyperlinks {
            domainResolver(foundHyperlink, resolvedUrlsInDomain)
        }
    }()

    go func() {
        foundUrls := make(map[string]struct{})
        for resolvedUrl := range resolvedUrlsInDomain {
            if _, ok := foundUrls[resolvedUrl]; !ok {
                fmt.Println(resolvedUrl)
                foundUrls[resolvedUrl] = struct{}{}
                go func() { 
                    unexploredUrls <- resolvedUrl
                }()
            }
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
            c <- removeFragmentIdentifier(domain + url)
        } else if strings.HasPrefix(url, domain) {
            c <- removeFragmentIdentifier(url)
        }
    }
}

func removeFragmentIdentifier(url string) string {
    return strings.Split(url, "#")[0]
}