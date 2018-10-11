package main


import (
    "fmt"
    "log"
    "net/http"
	"golang.org/x/net/html"
	"bytes"
    "io/ioutil"
    "regexp"
)

func everythingHandler(w http.ResponseWriter, r *http.Request) {
    // fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
    crawl("https://monzo.com/")
}

func main() {
    http.HandleFunc("/", everythingHandler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func crawl(root string) {
	var rootDomainFilter = domainFilter(root)

	resp, err := http.Get(root)
	defer resp.Body.Close()
	if err == nil {
		bodyAsByteArray, _ := ioutil.ReadAll(resp.Body)
		
		doc, _ := html.Parse(bytes.NewReader(bodyAsByteArray))

		printLinksRecursive(doc, rootDomainFilter)


	} else{
		fmt.Printf("ERROR: %s", err)
	}

}

func printLinksRecursive(n *html.Node, rootDomainFilter) {

    if n.Type == html.ElementNode && n.Data == "a" {
        for _, a := range n.Attr {
            if (a.Key == "href" && rootDomainFilter(a.Val)) {
                fmt.Println(a.Val)
                break
            }
        }
    }
    for c := n.FirstChild; c != nil; c = c.NextSibling {
        printLinksRecursive(c)
    }
}

func domainFilter(root string) func(link string) bool {
	return func(link string) {
		return strings.HasPrefix(link, "/") || strings.HasPrefix(link, root)
	}
}