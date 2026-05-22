package main

import (
	"reflect"
	"testing"
)

func TestPrependUnique(t *testing.T) {
	got := prependUnique("0.3.0", []string{"0.2.0", "0.1.0"})
	want := []string{"0.3.0", "0.2.0", "0.1.0"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestPrependUniqueRemovesDuplicate(t *testing.T) {
	got := prependUnique("0.2.0", []string{"0.3.0", "0.2.0", "0.1.0"})
	want := []string{"0.2.0", "0.3.0", "0.1.0"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
