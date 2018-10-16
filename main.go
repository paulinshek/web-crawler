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
    filterAndTransformLinks := filterOrResolve(domain)

    unexploredUrls := make(chan string)
    go func() {
        unexploredUrls <- domain // start
    }()

    foundHyperlinks := make(chan string)
    go exploreForLinks(unexploredUrls, foundHyperlinks)

    resolvedUrlsInDomain := make(chan string)
    go filterAndTransformLinks(foundHyperlinks, resolvedUrlsInDomain)


    go rerouteUnexploredLinks(resolvedUrlsInDomain, unexploredUrls)
}

func main() {
    http.HandleFunc("/", everythingHandler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func exploreForLinks(in chan string, out chan string) {
    for url := range in {
        resp, err := http.Get(url)
        defer resp.Body.Close()
        if err == nil {
            bodyAsByteArray, _ := ioutil.ReadAll(resp.Body)
            doc, _ := html.Parse(bytes.NewReader(bodyAsByteArray))

            pushHyperlinksToChannelRecursively(doc, out)

        } else{
            fmt.Println("ERROR: %s", err)
        }
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

func rerouteUnexploredLinks(in chan string, out chan string) {
    foundUrls := make(map[string]struct{})
    for resolvedUrl := range in {
        if _, ok := foundUrls[resolvedUrl]; !ok {
            fmt.Println(resolvedUrl)
            foundUrls[resolvedUrl] = struct{}{}
            go func() { 
                out <- resolvedUrl
            }()
        }
    }
}

func filterOrResolve(domain string) func(in chan string, out chan string) {
    return func(in chan string, out chan string) {
        for url := range in {
            if strings.HasPrefix(url, "/") {
                out <- removeFragmentIdentifier(domain + url)
            } else if strings.HasPrefix(url, domain) {
                out <- removeFragmentIdentifier(url)
            }
        }
    }
}

func removeFragmentIdentifier(url string) string {
    return strings.Split(url, "#")[0]
}


// BUG printed urls are not completely unique
// TODO parent link should also be pushed to channel so that we can create a graph from this Data
// TODO actually return something
// TODO refactor to use in/out channels