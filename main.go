package main


import (
    "fmt"
    "log"
    "net/http"
	"golang.org/x/net/html"
	"bytes"
    "io/ioutil"
)

func everythingHandler(w http.ResponseWriter, r *http.Request) {
    // fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
    crawl()
}

func main() {
    http.HandleFunc("/", everythingHandler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func crawl() {
	resp, err := http.Get("https://monzo.com/")
	defer resp.Body.Close()
	if err == nil {
		bodyAsByteArray, _ := ioutil.ReadAll(resp.Body)
		
		doc, _ := html.Parse(bytes.NewReader(bodyAsByteArray))

		var f func(*html.Node)
		f = func(n *html.Node) {
		    if n.Type == html.ElementNode && n.Data == "a" {
		        for _, a := range n.Attr {
		            if a.Key == "href" {
		                fmt.Println(a.Val)
		                break
		            }
		        }
		    }
		    for c := n.FirstChild; c != nil; c = c.NextSibling {
		        f(c)
		    }
		}
		f(doc)


	} else{
		fmt.Printf("ERROR: %s", err)
	}

}