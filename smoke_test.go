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

	actualDotOutput := startWebcrawler("http://localhost:8080")
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

	actualDotOutput := startWebcrawler("http://localhost:8080")
	actualDotOutput = strings.Replace(actualDotOutput, " ", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\n", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\t", "", -1)
	expectedDotOutput := "digraph{node[label=\"http://localhost:8080\"]n1;}"

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

	actualDotOutput := startWebcrawler("http://localhost:8080")
	actualDotOutput = strings.Replace(actualDotOutput, " ", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\n", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\t", "", -1)
	expectedDotOutput := "digraph{node[label=\"http://localhost:8080\"]n1;}"

	if expectedDotOutput != actualDotOutput {
		t.Errorf("Web crawler was incorrect, got: <%s>, want: <%s>.", actualDotOutput, expectedDotOutput)
	}

}
