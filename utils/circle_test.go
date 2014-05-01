package utils

import (
	"testing"
)

func TestCircle(t *testing.T) {
	c := NewCircle(2)

	if str := c.String(); str != "" {
		t.Error(`new string should be "", not "`, str, `"`)
	}

	c.Shift("hai")
	if str := c.String(); str != "hai" {
		t.Error(`first shift be "hai", not "`, str, `"`)
	}

	c.Shift("yay")
	if str := c.String(); str != "hai yay" {
		t.Error(`second shift be "hai yay", not "`, str, `"`)
	}

	c.Shift("blay")
	if str := c.String(); str != "yay blay" {
		t.Error(`third shift be "yay blay", not "`, str, `"`)
	}
}
