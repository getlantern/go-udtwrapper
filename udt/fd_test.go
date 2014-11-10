package udt

import (
	"fmt"
	"testing"
)

func TestSocketConstruct(t *testing.T) {
	a, err := ResolveUDTAddr("udt", ":1234")
	if err != nil {
		t.Fatal(err)
	}

	s, err := socket(a)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(s)
}
