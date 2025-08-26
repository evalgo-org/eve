package network

import (
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/dustin/go-humanize"

	eve "eve.evalgo.org/common"
)

type WriteCounter struct {
	Total uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

func (wc WriteCounter) PrintProgress() {
	eve.Logger.Info("\r", strings.Repeat(" ", 50))
	eve.Logger.Info("\rDownloading...", humanize.Bytes(wc.Total), "complete")
}

func DownloadFile(token string, url string, filepath string) error {
	out, err := os.Create(filepath + ".tmp")
	if err != nil {
		return err
	}
	defer out.Close()
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+token)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	counter := &WriteCounter{}
	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	if err != nil {
		return err
	}
	err = os.Rename(filepath+".tmp", filepath)
	if err != nil {
		return err
	}
	return nil
}
