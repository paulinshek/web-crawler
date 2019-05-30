package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestOnePage(t *testing.T) {
	h := http.NewServeMux()
	h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<html></html>")
	})
	go http.ListenAndServe(":8080", h)

	actualDotOutput := startWebcrawler("http://localhost:8080/", "http://localhost:8080").String()
	actualDotOutput = strings.Replace(actualDotOutput, " ", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\n", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\t", "", -1)
	expectedDotOutput := "digraph{n1[label=\"http://localhost:8080/\"];}"

	if expectedDotOutput != actualDotOutput {
		t.Errorf("Web crawler was incorrect, got: <%s>, want: <%s>.", actualDotOutput, expectedDotOutput)
	}
}

func TestOneTwoPages(t *testing.T) {
	h := http.NewServeMux()
	h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<a href=\"http://localhost:8080/another-page\">my link</a>")
	})
	h.HandleFunc("/another-page", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<html></html>")
	})
	go http.ListenAndServe(":8080", h)

	actualDotOutput := startWebcrawler("http://localhost:8080/", "http://localhost:8080").String()
	actualDotOutput = strings.Replace(actualDotOutput, " ", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\n", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\t", "", -1)
	expectedDotOutput := "digraph{n1[label=\"http://localhost:8080\"];n2[label=\"http://localhost:8080/another-page\"];n1->n2;}"

	if expectedDotOutput != actualDotOutput {
		t.Errorf("Web crawler was incorrect, got: <%s>, want: <%s>.", actualDotOutput, expectedDotOutput)
	}

}

func TestLoop(t *testing.T) {
	h := http.NewServeMux()
	h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<a href=\"http://localhost:8080/\">back to me</a>")
	})
	go http.ListenAndServe(":8080", h)

	actualDotOutput := startWebcrawler("http://localhost:8080/", "http://localhost:8080").String()
	actualDotOutput = strings.Replace(actualDotOutput, " ", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\n", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\t", "", -1)

	expectedDotOutput := "digraph{n1[label=\"http://localhost:8080\"];n1->n1;}"

	if expectedDotOutput != actualDotOutput {
		t.Errorf("Web crawler was incorrect, got: <%s>, want: <%s>.", actualDotOutput, expectedDotOutput)
	}
}
