package errs

import (
	"errors"
	"fmt"
	"testing"
)

func TestClosedErr_Error(t *testing.T) {
	closedErr := NewClosedErr("closed")
	fmt.Println(closedErr.Error())
}

func TestIsClosedErr(t *testing.T) {
	if !IsClosedErr(NewClosedErr("closed")) {
		t.Errorf("IsClosedErr() test-1 failed")
	}

	if IsClosedErr(errors.New("closed")) {
		t.Errorf("IsClosedErr() test-2 failed")
	}
}

func TestNewClosedErr(t *testing.T) {
	fmt.Println(NewClosedErr("closed"))
}
