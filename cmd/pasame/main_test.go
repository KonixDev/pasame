package main

import (
	"reflect"
	"testing"
)

func TestCleanArgs(t *testing.T) {
	got := cleanArgs([]string{"-psn_0_12345", "/a.jpg", "--algo", "/b c.mp4"})
	if !reflect.DeepEqual(got, []string{"/a.jpg", "/b c.mp4"}) {
		t.Fatalf("%v", got)
	}
	if len(cleanArgs(nil)) != 0 {
		t.Fatal("nil")
	}
}
