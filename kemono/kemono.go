package kemono

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Downloader interface {
	Download(<-chan FileWithIndex, Creator, Post) <-chan error
	Get(url string) (resp *http.Response, err error)
	WriteContent(Creator, Post, string) error
}

type Log interface {
	Printf(format string, v ...interface{})
	Print(s string)
}

type DefaultLog struct {
	log *log.Logger
}

func (d *DefaultLog) Printf(format string, v ...interface{}) {
	d.log.Printf(format, v...)
}

func (d *DefaultLog) Print(s string) {
	d.log.Print(s)
}

// Filter return true for continue, false for skip

type CreatorFilter func(i int, post Creator) bool

type PostFilter func(i int, post Post) bool

type AttachmentFilter func(i int, attachment File) bool

type Option func(*Kemono)

type Kemono struct {
	// kemono or coomer ...
	Site string
	//download Banner
	Banner bool
	// All Creator
	creators []Creator

	// Creator filter
	creatorFilters []CreatorFilter

	// Post filter map[creator(<service>:<id>)][]PostFilter
	postFilters map[string][]PostFilter

	// Attachment filter map[creator(<service>:<id>)][]AttachmentFilter
	attachmentFilters map[string][]AttachmentFilter

	// Select a specific creator
	// If not specified, all creators will be selected
	users []Creator

	// downloader
	Downloader Downloader

	log Log

	retry int

	retryInterval time.Duration
}

func NewKemono(options ...Option) *Kemono {
	k := &Kemono{
		Site:              "kemono",
		Banner:            true,
		postFilters:       make(map[string][]PostFilter),
		attachmentFilters: make(map[string][]AttachmentFilter),
		retry:             3,
		retryInterval:     5 * time.Second,
	}
	for _, option := range options {
		option(k)
	}
	// lazy initialize downloader
	if k.Downloader == nil {
		panic("Downloader is nil")
	}
	if k.log == nil {
		k.log = &DefaultLog{log: log.New(os.Stdout, "", log.LstdFlags)}
	}
	return k
}

// WithDomain Set Site
func WithDomain(web string) Option {
	return func(k *Kemono) {
		k.Site = web
	}
}

func WithBanner(banner bool) Option {
	return func(k *Kemono) {
		k.Banner = banner
	}
}

// Custom the creator list
func WithCreators(creators []Creator) Option {
	return func(k *Kemono) {
		k.creators = creators
	}
}

// WithUsers Select a specific creator, if not specified, all creators will be selected
func WithUsers(user ...Creator) Option {
	return func(k *Kemono) {
		for _, u := range user {
			exist := false
			for _, c := range k.users {
				if c.Service == u.Service && c.Id == u.Id {
					exist = true
					break
				}
			}
			if !exist {
				k.users = append(k.users, u)
			}
		}
	}
}

// WithUsersPair Select a specific creator, if not specified, all creators will be selected
func WithUsersPair(serviceIdPairs ...string) Option {
	return func(k *Kemono) {
		if len(serviceIdPairs)%2 != 0 {
			k.log.Printf("serviceIdPairs length must be even")
			return
		}
		for i := 0; i < len(serviceIdPairs); i += 2 {
			exist := false
			for _, c := range k.users {
				if c.Service == serviceIdPairs[i] && c.Id == serviceIdPairs[i+1] {
					exist = true
					break
				}
			}
			if !exist {
				k.users = append(k.users, NewCreator(serviceIdPairs[i], serviceIdPairs[i+1]))
			}
		}
	}
}

// SetDownloader set Downloader
func SetDownloader(downloader Downloader) Option {
	return func(k *Kemono) {
		k.Downloader = downloader
	}
}

// SetLog set log
func SetLog(log Log) Option {
	return func(k *Kemono) {
		k.log = log
	}
}

// SetRetry set retry
func SetRetry(retry int) Option {
	return func(k *Kemono) {
		k.retry = retry
	}
}

// SetRetryInterval set retry interval
func SetRetryInterval(retryInterval time.Duration) Option {
	return func(k *Kemono) {
		k.retryInterval = retryInterval
	}
}

// WithCreatorFilter Creator filter
func WithCreatorFilter(filter ...CreatorFilter) Option {
	return func(k *Kemono) {
		k.addCreatorFilter(filter...)
	}
}

// WithPostFilter Post filter
func WithPostFilter(filter ...PostFilter) Option {
	return func(k *Kemono) {
		k.addPostFilter(filter...)
	}
}

func WithUserPostFilter(creator Creator, filter ...PostFilter) Option {
	return func(k *Kemono) {
		k.addUserPostFilter(creator.PairString(), filter...)
	}
}

// WithAttachmentFilter Attachment filter
func WithAttachmentFilter(filter ...AttachmentFilter) Option {
	return func(k *Kemono) {
		k.addAttachmentFilter(filter...)
	}
}

func WithUserAttachmentFilter(creator Creator, filter ...AttachmentFilter) Option {
	return func(k *Kemono) {
		k.addUserAttachmentFilter(creator.PairString(), filter...)
	}
}

// Start fetch and download
func (k *Kemono) Start() error {
	// initialize the creators
	if len(k.creators) == 0 {
		// fetch creators from kemono
		cs, err := k.FetchCreators()
		if err != nil {
			return err
		}
		k.creators = cs
	}

	//find creators
	if len(k.users) != 0 {
		var creators []Creator
		for _, user := range k.users {
			c, ok := FindCreator(k.creators, user.Id, user.Service)
			if !ok {
				k.log.Printf("Creator %s:%s not found", user.Service, user.Id)
				continue
			}
			creators = append(creators, c)
		}
		k.users = creators
	} else {
		k.users = k.creators
	}

	// Filter selected creators
	k.users = k.FilterCreators(k.users)

	// start download
	k.log.Printf("Start download %d creators", len(k.users))
	for _, creator := range k.users {
		// fetch posts
		posts, err := k.FetchPosts(creator.Service, creator.Id)
		if err != nil {
			return err
		}
		// filter posts
		posts = k.FilterPosts(posts)

		// filter attachments
		for i, post := range posts {
			// download banner if banner is true or file is not image
			if (k.Banner || !isImage(filepath.Ext(post.File.Name))) && post.File.Path != "" {
				res := make([]File, len(post.Attachments)+1)
				copy(res[1:], post.Attachments)
				res[0] = post.File
				post.Attachments = res
			}
			posts[i].Attachments = k.FilterAttachments(fmt.Sprintf("%s:%s", post.Service, post.User), post.Attachments)
		}

		// download posts
		err = k.DownloadPosts(creator, posts)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k *Kemono) addCreatorFilter(filter ...CreatorFilter) {
	k.creatorFilters = append(k.creatorFilters, filter...)
}

func (k *Kemono) addPostFilter(filter ...PostFilter) {
	k.postFilters["*"] = append(k.postFilters["*"], filter...)
}

func (k *Kemono) addUserPostFilter(user string, filter ...PostFilter) {
	k.postFilters[user] = append(k.postFilters[user], filter...)
}

func (k *Kemono) addAttachmentFilter(filter ...AttachmentFilter) {
	k.attachmentFilters["*"] = append(k.attachmentFilters["*"], filter...)
}

func (k *Kemono) addUserAttachmentFilter(user string, filter ...AttachmentFilter) {
	k.attachmentFilters[user] = append(k.attachmentFilters[user], filter...)
}

func (k *Kemono) filterCreator(i int, creator Creator) bool {
	for _, filter := range k.creatorFilters {
		if !filter(i, creator) {
			return false
		}
	}
	return true
}

func (k *Kemono) filterPost(i int, post Post) bool {
	for _, filter := range k.postFilters["*"] {
		if !filter(i, post) {
			return false
		}
	}
	for _, filter := range k.postFilters[fmt.Sprintf("%s:%s", post.Service, post.User)] {
		if !filter(i, post) {
			return false
		}
	}
	return true
}

func (k *Kemono) filterAttachment(user string, i int, attachment File) bool {
	for _, filter := range k.attachmentFilters["*"] {
		if !filter(i, attachment) {
			return false
		}
	}
	for _, filter := range k.attachmentFilters[user] {
		if !filter(i, attachment) {
			return false
		}
	}
	return true
}

func (k *Kemono) FilterCreators(creators []Creator) []Creator {
	var filteredCreators []Creator
	for i, creator := range creators {
		if k.filterCreator(i, creator) {
			filteredCreators = append(filteredCreators, creator)
		}
	}
	return filteredCreators
}

func (k *Kemono) FilterPosts(posts []Post) []Post {
	var filteredPosts []Post
	for i, post := range posts {
		if k.filterPost(i, post) {
			filteredPosts = append(filteredPosts, post)
		}
	}
	return filteredPosts
}

func (k *Kemono) FilterAttachments(user string, attachments []File) []File {
	var filteredAttachments []File
	for i, attachment := range attachments {
		if k.filterAttachment(user, i, attachment) {
			filteredAttachments = append(filteredAttachments, attachment)
		}
	}
	return filteredAttachments
}

// ReleaseDateFilter A Post  filter that filters posts with release date
func ReleaseDateFilter(from, to time.Time) PostFilter {
	return func(i int, post Post) bool {
		return post.Published.After(from) && post.Published.Before(to)
	}
}

// ReleaseDateAfterFilter A Post  filter that filters posts with release date after
func ReleaseDateAfterFilter(from time.Time) PostFilter {
	return func(i int, post Post) bool {
		return post.Published.After(from)
	}
}

// ReleaseDateBeforeFilter A Post  filter that filters posts with release date before
func ReleaseDateBeforeFilter(to time.Time) PostFilter {
	return func(i int, post Post) bool {
		return post.Published.Before(to)
	}
}

// EditDateFilter A Post  filter that filters posts with edit date
func EditDateFilter(from, to time.Time) PostFilter {
	return func(i int, post Post) bool {
		return post.Edited.After(from) && post.Edited.Before(to)
	}
}

// EditDateAfterFilter A Post  filter that filters posts with edit date after
func EditDateAfterFilter(from time.Time) PostFilter {
	return func(i int, post Post) bool {
		return post.Edited.After(from)
	}
}

// EditDateBeforeFilter A Post  filter that filters posts with edit date before
func EditDateBeforeFilter(to time.Time) PostFilter {
	return func(i int, post Post) bool {
		return post.Edited.Before(to)
	}
}

func IdFilter(ids ...string) PostFilter {
	return func(i int, post Post) bool {
		for _, id := range ids {
			if id == post.Id {
				return true
			}
		}
		return false
	}
}

// NumbFilter A Post filter that filters posts with a specific number
func NumbFilter(f func(i int) bool) PostFilter {
	return func(i int, post Post) bool {
		return f(i)
	}
}

// ExtensionFilter A attachmentFilter filter that filters attachments with a specific extension
func ExtensionFilter(extension ...string) AttachmentFilter {
	return func(i int, attachment File) bool {
		ext := filepath.Ext(attachment.Name)
		for _, e := range extension {
			if ext == e {
				return true
			}
		}
		return false
	}
}

// ExtensionExcludeFilter A attachmentFilter filter that filters attachments without a specific extension
func ExtensionExcludeFilter(extension ...string) AttachmentFilter {
	return func(i int, attachment File) bool {
		ext := filepath.Ext(attachment.Name)
		for _, e := range extension {
			if ext == e {
				return false
			}
		}
		return true
	}
}
