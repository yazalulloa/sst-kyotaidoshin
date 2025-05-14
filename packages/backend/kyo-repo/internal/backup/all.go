package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/yaz/kyo-repo/internal/apartments"
	"github.com/yaz/kyo-repo/internal/buildings"
	"github.com/yaz/kyo-repo/internal/receipts"
	"github.com/yaz/kyo-repo/internal/util"
	"io"
	"os"
	"path/filepath"
	"sync"
)

func AllBackup(ctx context.Context, filename string) (string, error) {

	var wg sync.WaitGroup
	wg.Add(3)
	errorChan := make(chan error, 3)

	var apartmentsFilePath string
	var buildingsFilePath string
	var receiptsFilePath string

	go func() {
		defer wg.Done()
		path, err := apartments.NewService(ctx).Backup()

		if err != nil {
			errorChan <- err
			return
		}
		apartmentsFilePath = path
	}()

	go func() {
		defer wg.Done()
		path, err := buildings.NewService(ctx).Backup()

		if err != nil {
			errorChan <- err
			return
		}
		buildingsFilePath = path
	}()

	go func() {
		defer wg.Done()
		path, err := receipts.NewService(ctx).Backup()

		if err != nil {
			errorChan <- err
			return
		}
		receiptsFilePath = path
	}()

	wg.Wait()
	close(errorChan)

	files := []string{receiptsFilePath, apartmentsFilePath, buildingsFilePath}

	defer func() {
		for _, file := range files {
			err := os.Remove(file)
			if err != nil {
				fmt.Printf("Error deleting file %s: %s", file, err)
			}
		}
	}()

	err := util.HasErrors(errorChan)
	if err != nil {
		return "", err
	}

	path := filepath.Join("/tmp/", filename)
	out, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("error creating file %s", err)
	}

	defer out.Close()

	err = createArchive(files, out)
	if err != nil {
		return "", fmt.Errorf("error creating archive: %s", err)
	}

	return path, nil
}

func createArchive(files []string, buf io.Writer) error {
	// Create new Writers for gzip and tar
	// These writers are chained. Writing to the tar writer will
	// write to the gzip writer which in turn will write to
	// the "buf" writer
	gw := gzip.NewWriter(buf)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Iterate over files and add them to the tar archive
	for _, file := range files {
		err := addToArchive(tw, file)
		if err != nil {
			return err
		}
	}

	return nil
}

func addToArchive(tw *tar.Writer, filename string) error {
	// Open the file which will be written into the archive
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get FileInfo about our file providing file size, mode, etc.
	info, err := file.Stat()
	if err != nil {
		return err
	}

	// Create a tar Header from the FileInfo data
	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return err
	}

	// Use full path as name (FileInfoHeader only takes the basename)
	// If we don't do this the directory strucuture would
	// not be preserved
	// https://golang.org/src/archive/tar/common.go?#L626
	//header.Name = file.Name()

	// Write file header to the tar archive
	err = tw.WriteHeader(header)
	if err != nil {
		return err
	}

	// Copy file content to tar archive
	_, err = io.Copy(tw, file)
	if err != nil {
		return err
	}

	return nil
}
