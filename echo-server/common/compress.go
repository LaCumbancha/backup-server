package common

import (
	"os"
	"io"
	"strings"
	"archive/tar"
	"compress/gzip"

	log "github.com/sirupsen/logrus"
)

func TarAppender(filePath string, tarWriter *tar.Writer, fileInfo os.FileInfo) {
	file, err := os.Open(filePath)
	if err != nil {
	    log.Fatal("Error opening file %s.", filePath, err)
	}
	defer file.Close()

	header := new(tar.Header)
	header.Name = filePath
	header.Size = fileInfo.Size()
	header.Mode = int64(fileInfo.Mode())
	header.ModTime = fileInfo.ModTime()

	err = tarWriter.WriteHeader(header)
	if err != nil {
	    log.Fatal("Error writing Tar header for file %s.", filePath, err)
	}

	_, err = io.Copy(tarWriter, file)
	if err != nil {
	    log.Fatal("Error appending file %s content to tar file.", filePath, err)
	}
}

func IterativeCompression(dirPath string, tarWriter *tar.Writer) {
	dir, err := os.Open(dirPath)
	if err != nil {
	    log.Fatal("Error opening directory %s for backup.", dirPath, err)
	}
	defer dir.Close()

	filesInfo, err := dir.Readdir(0)
	if err != nil {
	    log.Fatal("Error reading directory %s for backup.", dirPath, err)
	}

	for _, fileInfo := range filesInfo {
	  fullPath := dirPath + "/" + fileInfo.Name()

	  if fileInfo.IsDir() {
	  	log.Debugf("Accessing new directory for compression in %s", fullPath)
	    IterativeCompression(fullPath, tarWriter)
	  } else {
	  	log.Debugf("Adding file %s to TarGz", fullPath)
	    TarAppender(fullPath, tarWriter, fileInfo)
	  }
	}
}

func GenerateBackupFile(outputName string, inPath string) {
	fileWriter, err := os.Create(outputName)
	if err != nil {
	    log.Fatal("Error creating fileWriter for compressor.", err)
	}
	defer fileWriter.Close()

	gzipWriter := gzip.NewWriter(fileWriter)
	if err != nil {
	    log.Fatal("Error creating gzipWriter for compressor.", err)
	}
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	if err != nil {
	    log.Fatal("Error creating tarWriter for compressor.", err)
	}
	defer tarWriter.Close()

	IterativeCompression(strings.TrimRight(inPath,"/"), tarWriter)
}
