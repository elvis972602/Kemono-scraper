package main

import (
	"fmt"
	"path/filepath"
	"time"
)

func main() {
	downloader := NewDownloader(
		BaseURL("https://kemono.party"),
		// the amount of download at the same time
		MaxConcurrent(3),
		Timeout(300*time.Second),
		// async download, download several files at the same time,
		// may cause the file order is not the same as the post order
		// you can use save path rule to control it
		Async(true),
		// the file will order by name in <order>-<file name>
		SavePath(func(creator Creator, post Post, i int, attachment File) string {
			var name string
			if filepath.Ext(attachment.Name) == ".zip" {
				name = attachment.Name
			} else {
				name = fmt.Sprintf("%d-%s", i, attachment.Name)
			}
			return fmt.Sprintf(filepath.Join("./download", "%s", "%s", "%s"), ValidDirectoryName(creator.Name), ValidDirectoryName(post.Title), ValidDirectoryName(name))
		}),
	)
	K := NewKemono(
		WithUsers("123456", "service", "654321", "service2"),
		WithBanner(false),
		WithPostFilter(
			ReleaseDateFilter(time.Now().AddDate(0, 0, -365), time.Now()),
		),
		WithAttachmentFilter(
			ExtensionFilter(".jpg", ".png", ".zip"),
		),
		SetDownloader(downloader),
	)
	K.Start()
}
