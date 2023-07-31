package downloader

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/elvis972602/kemono-scraper/kemono"
	"github.com/elvis972602/kemono-scraper/utils"
	"html/template"
	"io"
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

	cookies []*http.Cookie

	retry int

	retryInterval time.Duration

	content bool

	progress *Progress

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
				TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
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

	d.progress = NewProgress(d.log)
	d.progress.Run(100 * time.Millisecond)

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

// RateLimit limit the rate of download per second
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

func WithCookie(cookies []*http.Cookie) DownloadOption {
	return func(d *downloader) {
		d.cookies = cookies
	}
}

func WithProxy(proxy string) DownloadOption {
	return func(d *downloader) {
		AddProxy(proxy, d.client.Transport.(*http.Transport))
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
	ext := filepath.Ext(attachment.Name)
	if ext == "" {
		ext = filepath.Ext(attachment.Path)
	}
	if ext == ".zip" {
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

func WithContent(content bool) DownloadOption {
	return func(d *downloader) {
		d.content = content
	}
}

func (d *downloader) Get(url string) (resp *http.Response, err error) {
	var (
		req *http.Request
	)
	if req, err = newGetRequest(context.Background(), d.Header, d.cookies, url); err != nil {
		return
	}
	return d.client.Do(req)
}

func (d *downloader) WriteContent(creator kemono.Creator, post kemono.Post, content string) error {
	if !d.content {
		return nil
	}
	path := d.SavePath(creator, post, 0, kemono.File{Path: "content.html", Name: "content.html"})
	path = filepath.Join(filepath.Dir(path), "content.html")
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}
	file, err := os.Create(path)
	contentTemplate := `<!DOCTYPE html>
<html>
<head>
    <title>{{ .Title }}</title>
</head>
<body>
    {{ .Content }}
</body>
</html>`
	tmpl, err := template.New("content").Parse(contentTemplate)
	if err != nil {
		return err
	}
	return tmpl.Execute(file, struct {
		Title   string
		Content template.HTML
	}{
		Title:   post.Title,
		Content: template.HTML(content),
	})
}

func (d *downloader) Download(files <-chan kemono.FileWithIndex, creator kemono.Creator, post kemono.Post) <-chan error {
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
							errCh <- err
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
			d.log.Printf("file %s already exists, skip", filePath)
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
		//err = errors.New("download file error: " + err.Error())
		return err
	}
	time.Sleep(1 * time.Second)
	return nil
}

// download the file from the url, and save to the file
func (d *downloader) downloadFile(filePath, url string) error {
	d.reteLimiter.Token()

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout)
	defer cancel()

	req, err := newGetRequest(ctx, d.Header, d.cookies, url)
	if err != nil {
		return fmt.Errorf("new request error: %w", err)
	}

	var get func() error

	get = func() error {
		bar := NewProgressBar(fmt.Sprintf("%s", filepath.Base(filePath)), 0, 30)
		d.progress.AddBar(bar)
		defer func() {
			if !bar.IsDone() {
				d.progress.Failed(bar, fmt.Errorf("download failed"))
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
			d.progress.Cancel(bar, "size out of range")
			return nil
		}

		// 429 too many requests
		if resp.StatusCode == http.StatusTooManyRequests {
			d.progress.Failed(bar, fmt.Errorf("http 429"))
			return fmt.Errorf("request too many times")
		}

		if resp.StatusCode != http.StatusOK {
			d.progress.Failed(bar, fmt.Errorf("http %d", resp.StatusCode))
			return fmt.Errorf("failed to download file: %d", resp.StatusCode)
		}

		tmpFilePath := filePath + ".tmp"
		tmpFile, err := os.Create(tmpFilePath)
		if err != nil {
			// delete the tmp file
			_ = os.Remove(tmpFilePath)
			return fmt.Errorf("create tmp file error: %w", err)
		}

		defer func() {
			_ = tmpFile.Close()
			_ = os.Remove(tmpFilePath)
		}()

		_, err = io.Copy(io.MultiWriter(tmpFile, bar), resp.Body)
		if err != nil {
			d.progress.Failed(bar, err)
			return fmt.Errorf("io copy error: %w", err)
		}

		err = tmpFile.Close()
		if err != nil {
			return fmt.Errorf("close tmp file error: %w", err)
		}

		// rename the tmp file to the file
		err = os.Rename(tmpFilePath, filePath)
		if err != nil {
			return fmt.Errorf("rename file error: %w", err)
		}

		d.progress.Success(bar)
		return nil
	}

	for i := 0; i < d.retry; i++ {
		err = get()
		if err == nil {
			return nil
		}
		d.log.Printf("download failed: %s, retry after %.1f seconds...", err.Error(), d.retryInterval.Seconds())
		time.Sleep(d.retryInterval)
	}
	return fmt.Errorf("failed to download file: %w", err)

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
		defer file.Close()
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

func newGetRequest(ctx context.Context, header Header, cookies []*http.Cookie, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	// set headers
	for k, v := range header {
		req.Header.Set(k, v)
	}
	// set cookies
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return req, nil
}

func DirectoryName(p kemono.Post) string {
	return fmt.Sprintf("[%s] [%s] %s", p.Published.Format("20060102"), p.Id, p.Title)
}
