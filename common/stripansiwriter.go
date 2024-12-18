package common

import (
	"github.com/acarl005/stripansi"
	"io"
)

type stripAnsiWriter struct {
	io.Writer
}

func NewStripAnsiWriter(w io.Writer) io.Writer {
	return stripAnsiWriter{w}
}

func (w stripAnsiWriter) Write(p []byte) (n int, err error) {
	return w.Writer.Write([]byte(stripansi.Strip(string(p))))
}
