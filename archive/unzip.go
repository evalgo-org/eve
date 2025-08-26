package archive

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"

	eve "eve.evalgo.org/common"
)

func UnZip(zipPath string, tgtPath string) {
	eve.Logger.Info(zipPath, tgtPath)
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		panic(err)
	}
	defer archive.Close()
	for _, f := range archive.File {
		filePath := filepath.Join(tgtPath, f.Name)
		eve.Logger.Info("unzipping file ", filePath)
		if !strings.HasPrefix(filePath, filepath.Clean(tgtPath)+string(os.PathSeparator)) {
			eve.Logger.Info("invalid file path")
			return
		}
		if f.FileInfo().IsDir() {
			eve.Logger.Info("creating directory", filePath)
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			panic(err)
		}
		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			panic(err)
		}
		fileInArchive, err := f.Open()
		if err != nil {
			panic(err)
		}
		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			panic(err)
		}
		dstFile.Close()
		fileInArchive.Close()
	}
}
