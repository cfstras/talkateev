package utils

import (
	"strings"
)

type Circle struct {
	words  []string
	endPos int // after end
}

func NewCircle(num int) Circle {
	return Circle{make([]string, num), 0}
}

func (c *Circle) Shift(word string) {
	c.words[c.endPos] = word
	c.endPos++
	c.endPos %= len(c.words)
}

func app(to, slice []string) []string {
	for _, v := range slice {
		if v != "" {
			to = append(to, v)
		}
	}
	return to
}

func (c *Circle) String() string {
	str := make([]string, 0, len(c.words))

	str = app(str, c.words[c.endPos:])
	str = app(str, c.words[:c.endPos])

	return strings.Join(str, " ")
}
