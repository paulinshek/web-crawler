package main

import (
	"context"
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
	server := &http.Server{
		Addr:    ":8080",
		Handler: h,
	}
	defer server.Shutdown(context.Background())
	go server.ListenAndServe()

	actualDotOutput := startWebcrawler("http://localhost:8080/", "http://localhost:8080").String()
	actualDotOutput = strings.Replace(actualDotOutput, " ", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\n", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\t", "", -1)
	expectedDotOutput := "digraph{node[label=\"http://localhost:8080/\"]n1;}"

	if expectedDotOutput != actualDotOutput {
		t.Errorf("Web crawler was incorrect, got: <%s>, want: <%s>.", actualDotOutput, expectedDotOutput)

	}
}

func TestTwoPages(t *testing.T) {
	h := http.NewServeMux()
	h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<html><a href=\"http://localhost:8080/another-page\">my link</a></html>")
	})
	h.HandleFunc("/another-page", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<html></html>")
	})
	server := &http.Server{
		Addr:    ":8080",
		Handler: h,
	}
	defer server.Shutdown(context.Background())
	go server.ListenAndServe()

	actualDotOutput := startWebcrawler("http://localhost:8080/", "http://localhost:8080").String()
	actualDotOutput = strings.Replace(actualDotOutput, " ", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\n", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\t", "", -1)
	expectedDotOutput := "digraph{node[label=\"http://localhost:8080/\"]n1;node[label=\"http://localhost:8080/another-page\"]n2;n1->n2;}"
	alternativeExpectedDotOutput := "digraph{node[label=\"http://localhost:8080/another-page\"]n2;node[label=\"http://localhost:8080/\"]n1;n1->n2;}"

	if expectedDotOutput != actualDotOutput && alternativeExpectedDotOutput != actualDotOutput {
		t.Errorf("Web crawler was incorrect, got: <%s>, want: <%s> or : <%s>.", actualDotOutput, expectedDotOutput, alternativeExpectedDotOutput)
	}

}

func TestLoop(t *testing.T) {
	h := http.NewServeMux()
	h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<html><a href=\"http://localhost:8080/\">back to me</a></html>")
	})
	server := &http.Server{
		Addr:    ":8080",
		Handler: h,
	}
	defer server.Shutdown(context.Background())
	go server.ListenAndServe()

	actualDotOutput := startWebcrawler("http://localhost:8080/", "http://localhost:8080").String()
	actualDotOutput = strings.Replace(actualDotOutput, " ", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\n", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\t", "", -1)

	expectedDotOutput := "digraph{node[label=\"http://localhost:8080/\"]n1;n1->n1;}"

	if expectedDotOutput != actualDotOutput {
		t.Errorf("Web crawler was incorrect, got: <%s>, want: <%s>.", actualDotOutput, expectedDotOutput)
	}
}

func TestLinkToDifferentDomain(t *testing.T) {
	h := http.NewServeMux()
	h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<html><a href=\"http://different.domain/\">filter me</a></html>")
	})
	server := &http.Server{
		Addr:    ":8080",
		Handler: h,
	}
	defer server.Shutdown(context.Background())
	go server.ListenAndServe()

	actualDotOutput := startWebcrawler("http://localhost:8080/", "http://localhost:8080").String()
	actualDotOutput = strings.Replace(actualDotOutput, " ", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\n", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\t", "", -1)

	expectedDotOutput := "digraph{node[label=\"http://localhost:8080/\"]n1;}"

	if expectedDotOutput != actualDotOutput {
		t.Errorf("Web crawler was incorrect, got: <%s>, want: <%s>.", actualDotOutput, expectedDotOutput)
	}
}

func TestRelativeLink(t *testing.T) {
	h := http.NewServeMux()
	h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<html><a href=\"/another-page\">relative link</a></html>")
	})
	h.HandleFunc("/another-page", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<html></html>")
	})
	server := &http.Server{
		Addr:    ":8080",
		Handler: h,
	}
	defer server.Shutdown(context.Background())
	go server.ListenAndServe()

	actualDotOutput := startWebcrawler("http://localhost:8080/", "http://localhost:8080").String()
	actualDotOutput = strings.Replace(actualDotOutput, " ", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\n", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\t", "", -1)

	expectedDotOutput := "digraph{node[label=\"http://localhost:8080/\"]n1;node[label=\"http://localhost:8080/another-page\"]n2;n1->n2;}"
	if expectedDotOutput != actualDotOutput {
		t.Errorf("Web crawler was incorrect, got: <%s>, want: <%s>.", actualDotOutput, expectedDotOutput)
	}
}

func TestFragmentIdentifier(t *testing.T) {
	h := http.NewServeMux()
	h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<html><a href=\"http://localhost:8080/another-page#14254464\">relative link</a></html>")
	})
	h.HandleFunc("/another-page", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<html></html>")
	})
	server := &http.Server{
		Addr:    ":8080",
		Handler: h,
	}
	defer server.Shutdown(context.Background())
	go server.ListenAndServe()

	actualDotOutput := startWebcrawler("http://localhost:8080/", "http://localhost:8080").String()
	actualDotOutput = strings.Replace(actualDotOutput, " ", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\n", "", -1)
	actualDotOutput = strings.Replace(actualDotOutput, "\t", "", -1)

	expectedDotOutput := "digraph{node[label=\"http://localhost:8080/\"]n1;node[label=\"http://localhost:8080/another-page\"]n2;n1->n2;}"
	if expectedDotOutput != actualDotOutput {
		t.Errorf("Web crawler was incorrect, got: <%s>, want: <%s>.", actualDotOutput, expectedDotOutput)
	}
}