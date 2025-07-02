package api

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/yaz/kyo-repo/internal/util"
	"log"
	"os"
)

const (
	BACKUP_APARTMENTS_FILE = "apartments.json.gz"
	BACKUP_BUILDINGS_FILE  = "buildings.json.gz"
	BACKUP_RECEIPTS_FILE   = "receipts.json.gz"
	BACKUP_ALL_FILE        = "backup_all.tar.gz"
)

func Backup[T any](filename string, selectList func() ([]T, error)) (string, error) {
	filePath := util.TmpFileName(filename)
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("error creating file %s", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("Error closing file: %s", err)
		}
	}(file)

	// Create a buffered writer
	bufferedWriter := bufio.NewWriter(file)
	defer func(bufferedWriter *bufio.Writer) {
		err := bufferedWriter.Flush()
		if err != nil {
			log.Printf("Error flushing buffer: %s", err)
		}
	}(bufferedWriter) // Flush the buffer before closing

	// Create a gzip writer that writes to the buffered writer
	gzipWriter, err := gzip.NewWriterLevel(bufferedWriter, gzip.BestCompression)

	if err != nil {
		return "", fmt.Errorf("error creating gzip writer: %s", err)
	}

	defer func(gzipWriter *gzip.Writer) {
		err := gzipWriter.Close()
		if err != nil {
			log.Printf("Error closing gzip writer: %s", err)
		}
	}(gzipWriter)

	// Create a JSON encoder
	encoder := json.NewEncoder(gzipWriter)

	// Write the opening bracket for the JSON array
	_, err = gzipWriter.Write([]byte("["))
	if err != nil {
		return "", fmt.Errorf("error writing opening bracket to gzip writer: %s", err)
	}

	first := true

	for {

		items, err := selectList()
		if err != nil {
			return "", err
		}

		if len(items) == 0 {
			break
		}

		for _, item := range items {
			if !first {
				_, err = gzipWriter.Write([]byte(",")) // Add comma separator
				if err != nil {
					return "", fmt.Errorf("error writing comma to gzip writer: %s", err)
				}
			} else {
				first = false
			}

			if err := encoder.Encode(item); err != nil {
				return "", fmt.Errorf("encoding error: %s", err)
			}
		}
	}

	// Write the closing bracket for the JSON array
	_, err = gzipWriter.Write([]byte("]"))
	if err != nil {
		return "", fmt.Errorf("error writing closing bracket to gzip writer: %s", err)
	}

	return filePath, nil

}
