package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/elvis972602/kemono-scraper/downloader"
	"github.com/elvis972602/kemono-scraper/kemono"
	"github.com/elvis972602/kemono-scraper/term"
	"github.com/elvis972602/kemono-scraper/utils"
)

func main() {
	t := term.NewTerminal(os.Stdout, os.Stderr, false)

	d := downloader.NewDownloader(
		downloader.BaseURL("https://kemono.cr"),
		// the amount of download at the same time
		downloader.MaxConcurrent(3),
		downloader.Timeout(300*time.Second),
		// async download, download several files at the same time,
		// may cause the file order is not the same as the post order
		// you can use save path rule to control it
		downloader.Async(true),
		// the file will order by name in <order>-<file name>
		downloader.SavePath(func(creator kemono.Creator, post kemono.Post, i int, attachment kemono.File) string {
			var name string
			if filepath.Ext(attachment.Name) == ".zip" {
				name = attachment.Name
			} else {
				name = fmt.Sprintf("%d-%s", i, attachment.Name)
			}
			return fmt.Sprintf(filepath.Join("./download", "%s", "%s", "%s"), utils.ValidDirectoryName(creator.Name), utils.ValidDirectoryName(post.Title), utils.ValidDirectoryName(name))
		}),
		downloader.WithHeader(downloader.Header{
			"User-Agent":      downloader.UserAgent,
			"Referer":         "https://kemono.cr",
			"accept":          downloader.Accept,
			"accept-encoding": "gzip, deflate, br",
			"accept-language": "ja-JP;q=0.8,ja;q=0.7,en-US;q=0.6,en;q=0.5",
		}),
		downloader.RateLimit(2),
		downloader.Retry(5),
		downloader.RetryInterval(5*time.Second),
		downloader.SetLog(t),
	)
	user1 := kemono.NewCreator("service1", "123456")
	user2 := kemono.NewCreator("service2", "654321")
	K := kemono.NewKemono(
		kemono.WithUsers(user1, user2),
		kemono.WithUsersPair("service3", "987654"),
		kemono.WithBanner(true),
		kemono.WithPostFilter(
			kemono.ReleaseDateFilter(time.Now().AddDate(0, 0, -365), time.Now()),
		),
		kemono.WithAttachmentFilter(
			kemono.ExtensionFilter(".jpg", ".png", ".zip", ".gif"),
		),
		// a post filter for specific user
		kemono.WithUserPostFilter(user1, kemono.EditDateFilter(time.Now().AddDate(0, 0, -20), time.Now())),
		// an attachment filter for specific user
		kemono.WithUserAttachmentFilter(user2, func(i int, attachment kemono.File) bool {
			if i%2 == 0 {
				return false
			}
			return true
		}),
		kemono.SetDownloader(d),
		// if not set , use default log
		kemono.SetLog(t),
	)
	K.Start()
}
