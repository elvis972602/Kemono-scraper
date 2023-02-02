package main

import (
	"fmt"
	kemono "github.com/elvis972602/kemono-scraper"
	"path/filepath"
	"time"
)

func main() {
	downloader := kemono.NewDownloader(
		kemono.BaseURL("https://kemono.party"),
		// the amount of download at the same time
		kemono.MaxConcurrent(3),
		kemono.Timeout(300*time.Second),
		// async download, download several files at the same time,
		// may cause the file order is not the same as the post order
		// you can use save path rule to control it
		kemono.Async(true),
		// the file will order by name in <order>-<file name>
		kemono.SavePath(func(creator kemono.Creator, post kemono.Post, i int, attachment kemono.File) string {
			var name string
			if filepath.Ext(attachment.Name) == ".zip" {
				name = attachment.Name
			} else {
				name = fmt.Sprintf("%d-%s", i, attachment.Name)
			}
			return fmt.Sprintf(filepath.Join("./download", "%s", "%s", "%s"), kemono.ValidDirectoryName(creator.Name), kemono.ValidDirectoryName(post.Title), kemono.ValidDirectoryName(name))
		}),
		kemono.WithHeader(kemono.Header{
			"User-Agent":      kemono.UserAgent,
			"Referer":         "https://kemono.party",
			"accept":          kemono.Accept,
			"accept-encoding": "gzip, deflate, br",
			"accept-language": "ja-JP;q=0.8,ja;q=0.7,en-US;q=0.6,en;q=0.5",
		}),
		kemono.RateLimit(2),
		kemono.Retry(5),
	)
	K := kemono.NewKemono(
		kemono.WithUsers("74671556", "fanbox"),
		kemono.WithBanner(true),
		kemono.WithPostFilter(
			kemono.ReleaseDateFilter(time.Now().AddDate(0, 0, -365), time.Now()),
		),
		kemono.WithAttachmentFilter(
			kemono.ExtensionFilter(".jpg", ".png", ".zip", ".gif"),
		),
		kemono.SetDownloader(downloader),
	)
	K.Start()
}
