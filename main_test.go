package main

import (
	"testing"
)

func TestHandleUrl(t *testing.T) {

	results := make(chan UrlData, 10)
	var job Job = Job{Id: 1, Name: "https://habr.com/ru/post/215117/", Results: results}
	handleUrl(job)

	r := <-results
	if r.Size == 0 {
		t.Errorf("nok\tcheck hanleUrl execution, it returns: %#v\n", r)
	}
}
