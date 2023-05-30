package kemono

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/elvis972602/kemono-scraper/utils"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"time"
)

// FetchCreators fetch Creator list
func (k *Kemono) FetchCreators() (creators []Creator, err error) {
	k.log.Print("fetching creator list...")
	url := fmt.Sprintf("https://%s.party/api/creators", k.Site)
	resp, err := k.Downloader.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch creator list error: %s", err)
	}

	reader, err := handleCompressedHTTPResponse(resp)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("fetch creator list error: %s", err)
	}
	if k.Site == "kemono" {
		var c []KemonoCreator
		err = json.Unmarshal(data, &c)
		for _, v := range c {
			creators = append(creators, v.ToCreator())
		}
	} else if k.Site == "coomer" {
		var c []CoomerCreator
		err = json.Unmarshal(data, &c)
		for _, v := range c {
			creators = append(creators, v.ToCreator())
		}
	}
	if err != nil {
		return nil, fmt.Errorf("unmarshal creator list error: %s", err)
	}
	return
}

// FetchPosts fetch post list
func (k *Kemono) FetchPosts(service, id string) (posts []Post, err error) {
	url := fmt.Sprintf("https://%s.party/api/%s/user/%s", k.Site, service, id)
	perUnit := 50
	if k.Site == "coomer" {
		perUnit = 25
	}

	fetch := func(page int) (err error, finish bool) {
		k.log.Printf("fetching post list page %d...", page)
		purl := fmt.Sprintf("%s?o=%d", url, page*perUnit)

		retryCount := 0
		for retryCount < 3 {
			resp, err := k.Downloader.Get(purl)
			if err != nil {
				k.log.Printf("fetch post list error: %v", err)
				// Sleep for 5 seconds before retrying
				time.Sleep(5 * time.Second)
				retryCount++
				continue
			}

			if resp.StatusCode != http.StatusOK {
				k.log.Printf("fetch post list error: %s", resp.Status)
				time.Sleep(5 * time.Second)
				retryCount++
				continue
			}

			reader, err := handleCompressedHTTPResponse(resp)
			if err != nil {
				return err, false
			}

			data, err := ioutil.ReadAll(reader)
			if err != nil {
				return fmt.Errorf("fetch post list error: %s", err), false
			}
			reader.Close()

			var pr []PostRaw
			err = json.Unmarshal(data, &pr)
			if err != nil {
				return fmt.Errorf("unmarshal post list error: %s", err), false
			}
			if len(pr) == 0 {
				// final page
				return nil, true
			}
			for _, p := range pr {
				posts = append(posts, p.ParasTime())
			}
			return nil, false
		}

		return fmt.Errorf("fetch post list error: maximum retry count exceeded"), false
	}

	for i := 0; ; i++ {
		err, finish := fetch(i)
		if err != nil {
			return nil, err
		}
		if finish {
			break
		}
	}
	return
}

// DownloadPosts download posts
func (k *Kemono) DownloadPosts(creator Creator, posts []Post) (err error) {
	for _, post := range posts {
		k.log.Printf("download post: %s", utils.ValidDirectoryName(post.Title))
		if len(post.Attachments) == 0 {
			// no attachment
			continue
		}
		var (
			attachmentsChan = make(chan FileWithIndex, len(post.Attachments))
		)
		for _, a := range AddIndexToAttachments(post.Attachments) {
			attachmentsChan <- a
		}
		errChan := k.Downloader.Download(attachmentsChan, creator, post)
		for i := 0; i < len(errChan); i++ {
			err, ok := <-errChan
			if ok {
				k.log.Printf("download post error: %s", err)
				// TODO: record error...
			} else {
				break
			}
		}
	}
	return
}

func handleCompressedHTTPResponse(resp *http.Response) (io.ReadCloser, error) {
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		return reader, nil
	default:
		return resp.Body, nil
	}
}

func AddIndexToAttachments(attachments []File) []FileWithIndex {
	var files []FileWithIndex
	images := 0
	others := 0
	for _, a := range attachments {
		if isImage(a.Path) {
			files = append(files, a.Index(images))
			images++
		} else {
			files = append(files, a.Index(others))
			others++
		}
	}
	return files
}

func isImage(filename string) bool {
	switch filepath.Ext(filename) {
	case ".jpg", ".png", ".gif", ".webp", ".bmp", ".tiff", ".svg", ".ico", ".jpeg", ".jfif":
		return true
	default:
		return false
	}
}
