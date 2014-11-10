package udt

import (
	"testing"
)

func TestResolevUDTAddr(t *testing.T) {
	a, err := ResolveUDTAddr("udt", ":1234")
	if err != nil {
		t.Fatal(err)
	}

	if a.Network() != "udt" {
		t.Fatal("addr resolved incorrectly: %s", a.Network())
	}

	if a.String() != ":1234" {
		t.Fatal("addr resolved incorrectly: %s", a)
	}
}

func TestSocketConstruct(t *testing.T) {
	a, err := ResolveUDTAddr("udt", ":1234")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := socket(a); err != nil {
		t.Fatal(err)
	}
}

func TestSocketClose(t *testing.T) {
	a, err := ResolveUDTAddr("udt", ":1234")
	if err != nil {
		t.Fatal(err)
	}

	s, err := socket(a)
	if err != nil {
		t.Fatal(err)
	}

	if err := closeSocket(s); err != nil {
		t.Fatal(err)
	}

	if err := closeSocket(s); err == nil {
		t.Fatal("closing again did not produce error")
	}
}

func TestUdtFDConstruct(t *testing.T) {
	a, err := ResolveUDTAddr("udt", ":1234")
	if err != nil {
		t.Fatal(err)
	}

	s, err := socket(a)
	if err != nil {
		t.Fatal(err)
	}

	if int(s) <= 0 {
		t.Fatal("socket num invalid")
	}

	fd := newFD(s, a, "udt")
	if err := fd.Close(); err != nil {
		t.Fatal(err)
	}

	if int(fd.sock) != -1 {
		t.Fatal("sock should now be -1")
	}
}
