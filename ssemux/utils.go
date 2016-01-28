package ssemux

import (
	"bufio"
	"io"
)

func prefixLines(prefix string, r io.Reader) []byte {
	br := bufio.NewReader(r)
	var b []byte
	prependPrefix := true
	for true {
		line, isPrefix, err := br.ReadLine()
		if err != nil {
			break
		}
		if prependPrefix {
			b = append(b, []byte(prefix)...)
		}
		b = append(b, line...)
		if isPrefix {
			prependPrefix = false
		} else {
			b = append(b, []byte("\r\n")...)
			prependPrefix = true
		}
	}
	return b
}
