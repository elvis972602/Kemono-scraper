package kemono

import (
	"encoding/json"
	"fmt"
	"github.com/elvis972602/kemono-scraper/utils"
	"io/ioutil"
	"log"
	"net/http"
)

// FetchCreators fetch Creator list
func (k *Kemono) FetchCreators() (creators []Creator, err error) {
	k.log.Print("fetching creator list...")
	url := fmt.Sprintf("https://%s.party/api/creators", k.Site)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch creator list error: %s", err)
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
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

	fetch := func(page int) (err error, finish bool) {
		k.log.Printf("fetching post list page %d...", page)
		resp, err := http.Get(fmt.Sprintf("%s?o=%d", url, page*50))
		if err != nil {
			return fmt.Errorf("fetch post list error: %s", err), false
		}
		defer resp.Body.Close()
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("fetch post list error: %s", err), false
		}
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
		return
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
		for i, a := range post.Attachments {
			attachmentsChan <- a.Index(i)
		}
		errChan := k.Downloader.Download(attachmentsChan, creator, post)
		for i := 0; i < len(errChan); i++ {
			err, ok := <-errChan
			if ok {
				log.Println("download error: ", err)
				// TODO: record error...
			} else {
				break
			}
		}
	}
	return
}
