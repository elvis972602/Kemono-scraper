package kemono_scraper

import (
	"fmt"
	"log"
	"path/filepath"
	"time"
)

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

	// Post filter
	postFilters []PostFilter

	// Attachment filter
	attachmentFilters []AttachmentFilter

	// Select a specific creator
	// If not specified, all creators will be selected
	users []Creator

	// downloader
	Downloader Downloader
}

func NewKemono(options ...Option) *Kemono {
	k := &Kemono{
		Site:   "kemono",
		Banner: true,
	}
	for _, option := range options {
		option(k)
	}
	// lazy initialize downloader
	if k.Downloader == nil {
		k.Downloader = NewDownloader(
			BaseURL(fmt.Sprintf("https://%s.party", k.Site)),
			Async(true),
			RateLimit(2),
			WithHeader(Header{
				"User-Agent":      UserAgent,
				"Referer":         fmt.Sprintf("https://%s.party", k.Site),
				"accept-encoding": "gzip, deflate, br",
				"accept-language": "ja-JP;q=0.8,ja;q=0.7,en-US;q=0.6,en;q=0.5",
			}),
		)
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
func WithUsers(idServicePairs ...string) Option {
	return func(k *Kemono) {
		if len(idServicePairs)%2 != 0 {
			panic("idServicePairs must be even")
		}
		for i := 0; i < len(idServicePairs); i += 2 {
			k.users = append(k.users, Creator{
				Id:      idServicePairs[i],
				Service: idServicePairs[i+1],
			})
		}
		log.Printf("Select %d creators", len(k.users))
	}
}

// SetDownloader set Downloader
func SetDownloader(downloader Downloader) Option {
	return func(k *Kemono) {
		k.Downloader = downloader
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

// WithAttachmentFilter Attachment filter
func WithAttachmentFilter(filter ...AttachmentFilter) Option {
	return func(k *Kemono) {
		k.addAttachmentFilter(filter...)
	}
}

// Start fetch and download
func (k *Kemono) Start() {
	// initialize the creators
	if len(k.creators) == 0 {
		// fetch creators from kemono
		cs, err := k.FetchCreators()
		if err != nil {
			panic(err)
		}
		k.creators = cs
	}

	//find creators
	if len(k.users) != 0 {
		var exit []Creator
		for _, user := range k.users {
			c, ok := FindCreator(k.creators, user.Id, user.Service)
			if !ok {
				panic(fmt.Sprintf("user %s not found", user.Id))
			}
			exit = append(exit, c)
		}
		k.users = exit
	} else {
		k.users = k.creators
	}

	// Filter selected creators
	k.users = k.FilterCreators(k.users)

	// start download
	log.Printf("Start download %d creators", len(k.users))
	for _, creator := range k.users {
		// fetch posts
		posts, err := k.FetchPosts(creator.Service, creator.Id)
		if err != nil {
			panic(err)
		}
		// filter posts
		posts = k.FilterPosts(posts)

		// filter attachments
		for i, post := range posts {
			if k.Banner {
				res := make([]File, len(post.Attachments)+1)
				copy(res[1:], post.Attachments)
				res[0] = post.File
				post.Attachments = res
			}
			posts[i].Attachments = k.FilterAttachments(post.Attachments)
		}

		// download posts
		err = k.DownloadPosts(creator, posts)
		if err != nil {
			panic(err)
		}
	}
}

func (k *Kemono) addCreatorFilter(filter ...CreatorFilter) {
	k.creatorFilters = append(k.creatorFilters, filter...)
}

func (k *Kemono) addPostFilter(filter ...PostFilter) {
	k.postFilters = append(k.postFilters, filter...)
}

func (k *Kemono) addAttachmentFilter(filter ...AttachmentFilter) {
	k.attachmentFilters = append(k.attachmentFilters, filter...)
}

func (k *Kemono) filterCreator(i int, creator Creator) bool {
	for _, filter := range k.creatorFilters {
		if !filter(1, creator) {
			return false
		}
	}
	return true
}

func (k *Kemono) filterPost(i int, post Post) bool {
	for _, filter := range k.postFilters {
		if !filter(i, post) {
			return false
		}
	}
	return true
}

func (k *Kemono) filterAttachment(i int, attachment File) bool {
	for _, filter := range k.attachmentFilters {
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

func (k *Kemono) FilterAttachments(attachments []File) []File {
	var filteredAttachments []File
	for i, attachment := range attachments {
		if k.filterAttachment(i, attachment) {
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
