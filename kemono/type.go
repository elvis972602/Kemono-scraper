package kemono

import (
	"encoding/json"
	"fmt"
	"github.com/elvis972602/kemono-scraper/utils"
	"github.com/spf13/cast"
	"net/url"
	"path/filepath"
	"time"
)

type Timestamp struct {
	Time time.Time
}

func (t *Timestamp) UnmarshalJSON(b []byte) error {
	var timestamp float64
	err := json.Unmarshal(b, &timestamp)
	if err != nil {
		return err
	}

	t.Time = time.Unix(int64(timestamp), int64((timestamp-float64(int64(timestamp)))*1e9))
	return nil
}

type Creator struct {
	Favorited int       `json:"favorited"`
	Id        string    `json:"id"`
	Indexed   Timestamp `json:"indexed"`
	Name      string    `json:"name"`
	Service   string    `json:"service"`
	Updated   Timestamp `json:"updated"`
}

// GetID get creator id
func (c Creator) GetID() string {
	return c.Id
}

// GetService get creator Service
func (c Creator) GetService() string {
	return c.Service
}

func (c Creator) PairString() string {
	return fmt.Sprintf("%s:%s", c.Service, c.Id)
}

func NewCreator(service, id string) Creator {
	return Creator{
		Service: service,
		Id:      id,
	}
}

// FindCreator Get the Creator by ID and Service
func FindCreator(creators []Creator, id, service string) (Creator, bool) {
	for _, creator := range creators {
		if creator.Id == id && creator.Service == service {
			return creator, true
		}
	}
	return Creator{}, false
}

type File struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// GetURL return the url
func (f File) GetURL() string {
	ext := filepath.Ext(f.Name)
	name := f.Name[:len(f.Name)-len(ext)]
	return fmt.Sprintf("%s?f=%s%s", f.Path, url.QueryEscape(name), ext)
}

// GetHash get hash from file path
func (f File) GetHash() (string, error) {
	return utils.SplitHash(f.Path)
}

func (f File) Index(n int) FileWithIndex {
	return FileWithIndex{
		Index: n,
		File:  f,
	}
}

type FileWithIndex struct {
	Index int
	File
}

type PostRaw struct {
	Added       string      `json:"added"`
	Attachments []File      `json:"attachments"`
	Content     string      `json:"content"`
	Edited      string      `json:"edited"`
	Embed       interface{} `json:"embed"`
	File        File        `json:"file"`
	Id          string      `json:"id"`
	Published   string      `json:"published"`
	Service     string      `json:"service"`
	SharedFile  bool        `json:"shared_file"`
	Title       string      `json:"title"`
	User        string      `json:"user"`
}

func (p PostRaw) ParasTime() Post {
	var post Post
	post.Added, _ = cast.StringToDate(p.Added)
	post.Edited, _ = cast.StringToDate(p.Edited)
	post.Published, _ = cast.StringToDate(p.Published)
	post.Id = p.Id
	post.Service = p.Service
	post.Title = p.Title
	post.User = p.User
	post.Content = p.Content
	post.Embed = p.Embed
	post.SharedFile = p.SharedFile
	post.File = p.File
	post.Attachments = p.Attachments
	return post
}

type Post struct {
	Added       time.Time   `json:"added"`
	Attachments []File      `json:"attachments"`
	Content     string      `json:"content"`
	Edited      time.Time   `json:"edited"`
	Embed       interface{} `json:"embed"`
	File        File        `json:"file"`
	Id          string      `json:"id"`
	Published   time.Time   `json:"published"`
	Service     string      `json:"service"`
	SharedFile  bool        `json:"shared_file"`
	Title       string      `json:"title"`
	User        string      `json:"user"`
}

// User a creator according to the service and id
type User struct {
	Service string `json:"service"`
	Id      string `json:"id"`
}

// GetID get user id
func (u User) GetID() string {
	return u.Id
}

// GetService get user Service
func (u User) GetService() string {
	return u.Service
}

type FavoriteCreator struct {
	FavedSeq int    `json:"faved_seq"`
	Id       string `json:"id"`
	Index    string `json:"index"`
	Name     string `json:"name"`
	Service  string `json:"service"`
	Update   string `json:"update"`
}

var SiteMap = map[string]string{
	"patreon":       "kemono",
	"fanbox":        "kemono",
	"gumroad":       "kemono",
	"subscribestar": "kemono",
	"dlsite":        "kemono",
	"discord":       "kemono",
	"fantia":        "kemono",
	"boosty":        "kemono",
	"afdian":        "kemono",
	"onlyfans":      "coomer",
	"fansly":        "coomer",
}
