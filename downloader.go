package kemono_scraper

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
	rateLimit     = 2
	UserAgent     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"
	Accept        = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9"
)

type Header map[string]string

type Downloader interface {
	Download(<-chan FileWithIndex, Creator, Post) <-chan error
}

type DownloadOption func(*downloader)

type downloader struct {
	BaseURL string
	// Max concurrent download
	MaxConcurrent int
	// Max connection to the server
	MaxConnection int

	// Async download, download several files at the same time,
	// may cause the file order is not the same as the post order
	Async bool

	OverWrite bool

	// SavePath return the path to save the file
	SavePath func(creator Creator, post Post, i int, attachment File) string
	// timeout
	Timeout time.Duration

	reteLimiter *rateLimiter

	Header Header

	cookies chan []*http.Cookie

	retry int
}

func NewDownloader(options ...DownloadOption) Downloader {
	// with default options
	d := &downloader{
		MaxConcurrent: maxConcurrent,
		MaxConnection: maxConnection,
		SavePath:      defaultSavePath,
		Timeout:       300 * time.Second,
		Async:         false,
		reteLimiter:   newRateLimiter(rateLimit),
		retry:         2,
	}
	for _, option := range options {
		option(d)
	}
	if d.BaseURL == "" {
		panic("base url is empty")
	}
	if !d.Async {
		d.MaxConcurrent = 1
	}
	d.cookies = make(chan []*http.Cookie, d.MaxConcurrent)
	return d
}

// BaseURL set the base url
func BaseURL(baseURL string) DownloadOption {
	return func(d *downloader) {
		d.BaseURL = baseURL
	}
}

// MaxConcurrent set the max concurrent download
func MaxConcurrent(maxConcurrent int) DownloadOption {
	return func(d *downloader) {
		d.MaxConcurrent = maxConcurrent
	}
}

// Timeout set the timeout
func Timeout(timeout time.Duration) DownloadOption {
	return func(d *downloader) {
		d.Timeout = timeout
	}
}

// limit the rate of download per second
func RateLimit(n int) DownloadOption {
	return func(d *downloader) {
		d.reteLimiter = newRateLimiter(n)
	}
}

func WithHeader(header Header) DownloadOption {
	return func(d *downloader) {
		d.Header = header
	}
}

func SavePath(savePath func(creator Creator, post Post, i int, attachment File) string) DownloadOption {
	return func(d *downloader) {
		d.SavePath = savePath
	}
}

func defaultSavePath(creator Creator, post Post, i int, attachment File) string {
	return fmt.Sprintf(filepath.Join("./download", "%s", "%s", "%s"), ValidDirectoryName(creator.Name), ValidDirectoryName(post.Title), ValidDirectoryName(attachment.Name))
}

// Async set the async download option
func Async(async bool) DownloadOption {
	return func(d *downloader) {
		d.Async = async
	}
}

// OverWrite set the overwrite option
func OverWrite(overwrite bool) DownloadOption {
	return func(d *downloader) {
		d.OverWrite = overwrite
	}
}

func Retry(retry int) DownloadOption {
	return func(d *downloader) {
		d.retry = retry
	}
}

func (d *downloader) Download(files <-chan FileWithIndex, creator Creator, post Post) <-chan error {

	//TODO: implement download
	var (
		wg    sync.WaitGroup
		errCh = make(chan error, len(files))
	)

	for i := 0; i < d.MaxConcurrent; i++ {
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
						savePath := d.SavePath(creator, post, file.Index, file.File)
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
	var (
		f        *os.File
		complete bool
		err      error
	)
	if !d.OverWrite {
		f, complete, err = checkFileExitAndComplete(filePath, fileHash)
		defer f.Close()
		if err != nil {
			err = errors.New("check file error: " + err.Error())
			return err
		}
		if complete {
			log.Printf("file %s already exists, skip", filePath)
			return nil
		}
	} else {
		f, err = os.Create(filePath)
		defer f.Close()
		if err != nil {
			err = errors.New("create file error: " + err.Error())
			return err
		}
	}
	// download the file

	if err := d.downloadFile(f, url); err != nil {
		err = errors.New("download file error: " + err.Error())
		return err
	}
	time.Sleep(1 * time.Second)
	return nil
}

// download the file from the url, and save to the file
func (d *downloader) downloadFile(file *os.File, url string) error {
	d.reteLimiter.Token()

	log.Printf("downloading file %s", url)
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout)
	defer cancel()

	req, err := newGetRequest(ctx, d.Header, url)
	if err != nil {
		return fmt.Errorf("new request error: %w", err)
	}

	if len(d.cookies) > 0 {
		c, ok := <-d.cookies
		if ok {
			for _, cookie := range c {
				req.AddCookie(cookie)
			}
		}
	}

	var get func(retry int) error

	get = func(retry int) error {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to make request: %w", err)
		}
		defer resp.Body.Close()

		// 429 too many requests
		if resp.StatusCode == http.StatusTooManyRequests {
			if retry > 0 {
				log.Printf("request too many times, retry after 10 seconds...")
				time.Sleep(10 * time.Second)
				return get(retry - 1)
			} else {
				return fmt.Errorf("failed to download file: %d", resp.StatusCode)
			}
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to download file: %d", resp.StatusCode)
		}

		if len(resp.Cookies()) < d.MaxConcurrent {
			d.cookies <- resp.Cookies()
		}

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		return nil
	}

	return get(d.retry)

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

func newGetRequest(ctx context.Context, header Header, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	// set headers
	for k, v := range header {
		req.Header.Set(k, v)
	}
	return req, nil
}
