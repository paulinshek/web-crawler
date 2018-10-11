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
    crawl("https://monzo.com")
}

func main() {
    http.HandleFunc("/", everythingHandler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func crawl(root string) {
	linkExtractorDomainOnly := getFullUrlInRoot(root)


	resp, err := http.Get(root)
	defer resp.Body.Close()
	if err == nil {
		bodyAsByteArray, _ := ioutil.ReadAll(resp.Body)
		
		doc, _ := html.Parse(bytes.NewReader(bodyAsByteArray))

		printLinksRecursive(doc, linkExtractorDomainOnly)


	} else{
		fmt.Printf("ERROR: %s", err)
	}

}

func printLinksRecursive(n *html.Node, urlExtractor func(link string) (fullUrl string, err string) ) {

    if n.Type == html.ElementNode && n.Data == "a" {
        for _, a := range n.Attr {
            if a.Key == "href" {
                fullUrl, externalLinkErr := urlExtractor(a.Val)
                if (externalLinkErr == "") {
                	fmt.Println(fullUrl)
                }
                break
            }
        }
    }
    for c := n.FirstChild; c != nil; c = c.NextSibling {
        printLinksRecursive(c, urlExtractor)
    }
}

func getFullUrlInRoot(root string) func(link string) (fullUrl string, err string) {
	return func(link string) (fullUrl string, err string) {
		if strings.HasPrefix(link, "/")  {
			return root + link, ""
		} else if strings.HasPrefix(link, root) {
			return link, ""
		}
		return "", "error: external link"
	}
}