package common

import (
	"bytes"
	"github.com/sirupsen/logrus"
	"os"
)

type OutputSplitter struct{}

func (splitter *OutputSplitter) Write(p []byte) (n int, err error) {
	if bytes.Contains(p, []byte("level=error")) {
		return os.Stderr.Write(p)
	}
	return os.Stdout.Write(p)
}

var Logger = logrus.New()
