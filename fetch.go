package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// FetchCreators fetch Creator list
func (k *Kemono) FetchCreators() (creators []Creator, err error) {
	log.Printf("fetching creator list...")
	url := fmt.Sprintf("https://%s.party/api/creators", k.site)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch creator list error: %s", err)
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fetch creator list error: %s", err)
	}

	err = json.Unmarshal(data, &creators)
	if err != nil {
		return nil, fmt.Errorf("unmarshal creator list error: %s", err)
	}
	return
}

// FetchPosts fetch post list
func (k *Kemono) FetchPosts(service, id string) (posts []Post, err error) {
	url := fmt.Sprintf("https://%s.party/api/%s/user/%s", k.site, service, id)

	fetch := func(page int) (err error, finish bool) {
		log.Printf("fetching post list page %d...", page)
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
		log.Println("download post: ", post.Title)
		var (
			attachmentsChan = make(chan File, len(post.Attachments))
		)
		for _, a := range post.Attachments {
			attachmentsChan <- a
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
