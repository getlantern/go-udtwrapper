package udt

import (
	"testing"
)

func TestStuff(t *testing.T) {
	s, err := Dial("ip4", "127.0.0.1:11000")
	if err != nil {
		t.Errorf("Unable to dial: %s", err)
	} else {
		t.Logf("Socket is: %s", s)
	}
}
