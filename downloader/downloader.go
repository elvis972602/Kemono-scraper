package downloader

import (
	"context"
	"errors"
	"fmt"
	"github.com/elvis972602/kemono-scraper/kemono"
	"github.com/elvis972602/kemono-scraper/utils"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

const (
	maxConcurrent           = 5
	maxConnection           = 100
	rateLimit               = 2
	UserAgent               = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"
	Accept                  = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"
	AcceptEncoding          = "gzip, deflate, br"
	AcceptLanguage          = "en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7"
	SecChUA                 = "\"Google Chrome\";v=\"111\", \"Not(A:Brand\";v=\"8\", \"Chromium\";v=\"111\""
	SecChUAMobile           = "?0"
	SecFetchDest            = "document"
	SecFetchMode            = "navigate"
	SecFetchSite            = "none"
	SecFetchUser            = "?1"
	UpgradeInsecureRequests = "1"
)

type Log interface {
	Printf(format string, v ...interface{})
	Print(s string)
	SetStatus(s []string)
}

type Header map[string]string

type DownloadOption func(*downloader)

type downloader struct {
	BaseURL string
	// Max concurrent download
	MaxConcurrent int

	// Async download, download several files at the same time,
	// may cause the file order is not the same as the post order
	Async bool

	OverWrite bool

	maxSize int64

	minSize int64

	// SavePath return the path to save the file
	SavePath func(creator kemono.Creator, post kemono.Post, i int, attachment kemono.File) string
	// timeout
	Timeout time.Duration

	reteLimiter *utils.RateLimiter

	Header Header

	retry int

	retryInterval time.Duration

	progressBar *utils.ProgressBar

	log Log

	client *http.Client
}

func NewDownloader(options ...DownloadOption) kemono.Downloader {
	// with default options
	d := &downloader{
		MaxConcurrent: maxConcurrent,
		SavePath:      defaultSavePath,
		Timeout:       300 * time.Second,
		Async:         false,
		OverWrite:     false,
		maxSize:       1<<63 - 1,
		minSize:       0,
		reteLimiter:   utils.NewRateLimiter(rateLimit),
		retry:         2,
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:          maxConnection,
				MaxConnsPerHost:       maxConnection,
				MaxIdleConnsPerHost:   maxConnection,
				ResponseHeaderTimeout: 30 * time.Second,
			},
		},
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
	if d.log == nil {
		panic("log is nil")
	}

	d.progressBar = utils.NewProgressBar(d.log)

	go func() {
		tick := time.NewTicker(100 * time.Millisecond)
		for {
			select {
			case <-tick.C:
				d.progressBar.SetStatus()
			}
		}
	}()
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

// MaxSize set the max size of the file to download
func MaxSize(maxSize int64) DownloadOption {
	return func(d *downloader) {
		d.maxSize = maxSize
	}
}

// MinSize set the min size of the file to download
func MinSize(minSize int64) DownloadOption {
	return func(d *downloader) {
		d.minSize = minSize
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
		d.reteLimiter = utils.NewRateLimiter(n)
	}
}

func WithHeader(header Header) DownloadOption {
	return func(d *downloader) {
		d.Header = header
	}
}

func SavePath(savePath func(creator kemono.Creator, post kemono.Post, i int, attachment kemono.File) string) DownloadOption {
	return func(d *downloader) {
		d.SavePath = savePath
	}
}

// SetLog set the log
func SetLog(log Log) DownloadOption {
	return func(d *downloader) {
		d.log = log
	}
}

func defaultSavePath(creator kemono.Creator, post kemono.Post, i int, attachment kemono.File) string {
	var name string
	if filepath.Ext(attachment.Path) == ".zip" {
		name = attachment.Name
	} else {
		name = filepath.Base(attachment.Path)
	}
	return fmt.Sprintf(filepath.Join("./download", "%s", "%s", "%s"), utils.ValidDirectoryName(creator.Name), utils.ValidDirectoryName(DirectoryName(post)), utils.ValidDirectoryName(name))
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

func RetryInterval(interval time.Duration) DownloadOption {
	return func(d *downloader) {
		d.retryInterval = interval
	}
}

func (d *downloader) Download(files <-chan kemono.FileWithIndex, creator kemono.Creator, post kemono.Post) <-chan error {

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
		complete bool
		err      error
	)
	if !d.OverWrite {
		complete, err = checkFileExitAndComplete(filePath, fileHash)
		if err != nil {
			err = errors.New("check file error: " + err.Error())
			return err
		}
		if complete {
			d.log.Printf(utils.ShortenString("file ", filePath, " already exists, skip"))
			return nil
		}
	}

	err = os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	if err != nil {
		err = errors.New("create directory error: " + err.Error())
		return err
	}
	// download the file
	if err := d.downloadFile(filePath, url); err != nil {
		err = errors.New("download file error: " + err.Error())
		return err
	}
	time.Sleep(1 * time.Second)
	return nil
}

// download the file from the url, and save to the file
func (d *downloader) downloadFile(filePath, url string) error {
	d.reteLimiter.Token()

	//progressBar.Printf("downloading file %s", url)
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout)
	defer cancel()

	req, err := newGetRequest(ctx, d.Header, url)
	if err != nil {
		return fmt.Errorf("new request error: %w", err)
	}

	var get func(retry int) error

	get = func(retry int) error {
		bar := &utils.Bar{Since: time.Now(), Prefix: "Download", Content: fmt.Sprintf("%s", filepath.Base(filePath)), Max: 0, Length: 30}
		d.progressBar.AddBar(bar)
		defer func() {
			if !bar.IsDone() {
				d.progressBar.Cancel(bar, "Download failed")
			}
		}()

		resp, err := d.client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to make request: %w", err)
		}
		defer resp.Body.Close()

		// get content length
		contentLength, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
		if err != nil {
			return fmt.Errorf("failed to get content length: %w", err)
		}
		bar.Max = contentLength

		if contentLength > d.maxSize || contentLength < d.minSize {

			d.progressBar.Cancel(bar, fmt.Sprintf("%s out of range", utils.FormatSize(contentLength)))
			return nil
		}

		tmpFilePath := filePath + ".tmp"
		tmpFile, err := os.Create(tmpFilePath)
		if err != nil {
			// 删除临时文件
			_ = os.Remove(tmpFilePath)
			return fmt.Errorf("create tmp file error: %w", err)
		}
		defer func() {
			_ = tmpFile.Close()
		}()

		// 429 too many requests
		if resp.StatusCode == http.StatusTooManyRequests {
			d.progressBar.Cancel(bar, "http 429")
			if retry > 0 {
				d.log.Printf("request too many times, retry after %.1f seconds...", d.retryInterval.Seconds())
				time.Sleep(d.retryInterval)
				return get(retry - 1)
			} else {
				return fmt.Errorf("failed to download file: %d", resp.StatusCode)
			}
		}

		if resp.StatusCode != http.StatusOK {
			d.progressBar.Failed(bar, fmt.Errorf("http %d", resp.StatusCode))
			return fmt.Errorf("failed to download file: %d", resp.StatusCode)
		}

		_, err = utils.Copy(tmpFile, resp.Body, bar)
		if err != nil {
			d.progressBar.Failed(bar, err)
			return fmt.Errorf("failed to write file: %w", err)
		}

		// rename the tmp file to the file
		_ = tmpFile.Close()
		// 重命名文件
		err = os.Rename(tmpFilePath, filePath)
		if err != nil {
			return fmt.Errorf("rename file error: %w", err)
		}

		d.progressBar.Success(bar)
		return nil
	}

	return get(d.retry)

}

// check if the file exists, if exists, check if the file is complete,and return the file
// if the file is complete, return true
func checkFileExitAndComplete(filePath, fileHash string) (complete bool, err error) {
	// check if the file exists
	var (
		h    []byte
		file *os.File
	)
	f, err := os.Stat(filePath)
	if err != nil {
		// un exists
		return false, nil
	} else if f != nil {
		// file exists, check if the file is complete
		file, err = os.OpenFile(filePath, os.O_RDWR, 0644)
		if err != nil {
			err = fmt.Errorf("open file error: %w", err)
			return
		}
		h, err = utils.Hash(file)
		if err != nil {
			err = fmt.Errorf("get file hash error: %w", err)
			return
		}
		// check if the file is complete
		if fmt.Sprintf("%x", h) == fileHash {
			complete = true
			return
		}
	}
	return false, nil
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

func DirectoryName(p kemono.Post) string {
	return fmt.Sprintf("[%s] [%s] %s", p.Published.Format("20060102"), p.Id, p.Title)
}
