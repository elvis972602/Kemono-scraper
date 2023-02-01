package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	maxConcurrent = 5
	maxConnection = 16
)

type Downloader interface {
	Download(<-chan File, Creator, Post) <-chan error
}

type DownloadOptions func(*downloader)

type downloader struct {
	BaseURL string
	// Max concurrent download
	MaxConcurrent int
	// Max connection to the server
	MaxConnection int

	// Async download, download several files at the same time,
	// may cause the file order is not the same as the post order
	Async bool

	// SavePath return the path to save the file
	SavePath func(creator Creator, post Post, attachment File) string
	// timeout
	Timeout time.Duration

	AllowType []string
}

func NewDownloader(options ...DownloadOptions) Downloader {
	// with default options
	d := &downloader{
		MaxConcurrent: maxConcurrent,
		MaxConnection: maxConnection,
		SavePath:      defaultSavePath,
		Timeout:       300 * time.Second,
		Async:         true,
	}
	for _, option := range options {
		option(d)
	}
	return d
}

// BaseURL set the base url
func BaseURL(baseURL string) DownloadOptions {
	return func(d *downloader) {
		d.BaseURL = baseURL
	}
}

// MaxConcurrent set the max concurrent download
func MaxConcurrent(maxConcurrent int) DownloadOptions {
	return func(d *downloader) {
		d.MaxConcurrent = maxConcurrent
	}
}

func SavePath(savePath func(creator Creator, post Post, attachment File) string) DownloadOptions {
	return func(d *downloader) {
		d.SavePath = savePath
	}
}

func defaultSavePath(creator Creator, post Post, attachment File) string {
	return fmt.Sprintf(filepath.Join("./download", "%s", "%s", "%s"), ValidDirectoryName(creator.Name), ValidDirectoryName(post.Title), ValidDirectoryName(attachment.Name))
}

// Async set the async download option
func Async(async bool) DownloadOptions {
	return func(d *downloader) {
		d.Async = async
	}
}

func (d *downloader) Download(files <-chan File, creator Creator, post Post) <-chan error {

	//TODO: implement download
	var (
		wg            sync.WaitGroup
		maxConcurrent = 1
		errCh         = make(chan error, len(files))
	)
	if d.Async {
		maxConcurrent = d.MaxConcurrent
	}

	for i := 0; i < maxConcurrent; i++ {
		wg.Add(1)
		go func() {
			for {
				select {

				case file, ok := <-files:
					// download file
					if ok {
						url := d.BaseURL + file.GetURL()
						hash, err := file.GetHash()
						if err != nil {
							hash = ""
						}
						savePath := d.SavePath(creator, post, file)
						err = os.MkdirAll(filepath.Dir(savePath), os.ModePerm)
						if err != nil {
							errCh <- errors.New("create directory error: " + err.Error())
							continue
						}
						if err := d.download(savePath, url, hash); err != nil {
							errCh <- errors.New("download file error: " + err.Error())
							continue
						}
					}
				default:
					if len(files) == 0 {
						wg.Done()
						return
					}
				}
			}
		}()
	}
	wg.Wait()
	return errCh
}

// download downloads the file from the url
func (d *downloader) download(filePath, url, fileHash string) error {
	// check if the file exists
	f, complete, err := checkFileExitAndComplete(filePath, fileHash)
	defer f.Close()
	if err != nil {
		err = errors.New("check file error: " + err.Error())
		return err
	}
	if complete {
		log.Printf("file %s already exists, skip", filePath)
		return nil
	}
	// download the file

	if err = d.downloadFile(f, url); err != nil {
		err = errors.New("download file error: " + err.Error())
		return err
	}
	return nil
}

// download the file from the url, and save to the file
func (d *downloader) downloadFile(file *os.File, url string) error {
	log.Printf("downloading file %s", url)
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %d", resp.StatusCode)
	}

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil

}

// check if the file exists, if exists, check if the file is complete,and return the file
// if the file is complete, return true
func checkFileExitAndComplete(filePath, fileHash string) (file *os.File, complete bool, err error) {
	// check if the file exists
	var h []byte
	f, err := os.Stat(filePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			err = fmt.Errorf("check file error: %w", err)
			return
		} else {
			// create the file
			file, err = os.Create(filePath)
			if err != nil {
				err = fmt.Errorf("create file error: %w", err)
				return
			}
		}
	} else if f != nil {
		// file exists, check if the file is complete
		file, err = os.OpenFile(filePath, os.O_RDWR, 0644)
		if err != nil {
			err = fmt.Errorf("open file error: %w", err)
			return
		}
		h, err = Hash(file)
		if err != nil {
			err = fmt.Errorf("get file hash error: %w", err)
			return
		}
		// check if the file is complete
		if fmt.Sprintf("%x", h) == fileHash {
			complete = true
			return
		}
		err = file.Truncate(0)
		if err != nil {
			log.Printf("truncate file error: %s", err)
			return nil, false, err
		}
		_, err := file.Seek(0, 0)
		if err != nil {
			return nil, false, err
		}
	}
	return file, false, nil
}
