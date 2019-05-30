package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestOnePage(t *testing.T) {
	h := http.NewServeMux()
	h.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<a href=\"http://localhost:8080/test/another-page\">my link</a>")
	})
	h.HandleFunc("/another-page", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<a href=\"http://localhost:8080/test\">my link</a>")
	})
	go http.ListenAndServe(":8080", h)

	actualDotOutput := startWebcrawler("http://localhost:8080/", "http://localhost:8080").String()
	actualDotOutput = strings.Replace(actualDotOutput, " ", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\n", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\t", "", -1)
	expectedDotOutput := "digraph{node[label=\"http://localhost:8080\"]n1;}"

	if expectedDotOutput != actualDotOutput {
		t.Errorf("Web crawler was incorrect, got: <%s>, want: <%s>.", actualDotOutput, expectedDotOutput)
	}
}

func TestOneTwoPages(t *testing.T) {
	h := http.NewServeMux()
	h.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<a href=\"http://localhost:8080/test/another-page\">my link</a>")
	})
	h.HandleFunc("/another-page", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<a href=\"http://localhost:8080/test\">my link</a>")
	})
	go http.ListenAndServe(":8080", h)

	actualDotOutput := startWebcrawler("http://localhost:8080/test", "http://localhost:8080").String()
	actualDotOutput = strings.Replace(actualDotOutput, " ", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\n", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\t", "", -1)
	expectedDotOutput := "digraph{node[label=\"http://localhost:8080/test\"]n1;node[label=\"http://localhost:8080/test/another-page\"]n2;n1->n2;}"

	if expectedDotOutput != actualDotOutput {
		t.Errorf("Web crawler was incorrect, got: <%s>, want: <%s>.", actualDotOutput, expectedDotOutput)
	}

}

func TestOneTwoPagesWithLoop(t *testing.T) {
	h := http.NewServeMux()
	h.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<a href=\"http://localhost:8080/test/another-page\">my link</a>")
	})
	h.HandleFunc("/another-page", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<a href=\"http://localhost:8080/test\">my link</a>")
	})
	go http.ListenAndServe(":8080", h)

	actualDotOutput := startWebcrawler("http://localhost:8080/test", "http://localhost:8080").String()
	actualDotOutput = strings.Replace(actualDotOutput, " ", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\n", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\t", "", -1)

	if !strings.Contains(actualDotOutput, "node[label=\"http://localhost:8080/test\"]") {
		t.Errorf("Web crawler did not create node[label=\"http://localhost:8080/test\"]")
	}
	if !strings.Contains(actualDotOutput, "node[label=\"http://localhost:8080/test/another-page\"]") {
		t.Errorf("Web crawler did not create node[label=\"http://localhost:8080/test/another-page\"]")
	}
	if !strings.Contains(actualDotOutput, "n1->n2;") {
		t.Errorf("Web crawler did not create link between the two nodes")
	}
	if strings.Contains(actualDotOutput, "n3") {
		t.Errorf("Web crawler found too many nodes")
	}
}
