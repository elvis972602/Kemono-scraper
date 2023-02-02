package kemono_scraper

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
		WithHeader(Header{
			"User-Agent":      UserAgent,
			"Referer":         "https://kemono.party",
			"accept":          Accept,
			"accept-encoding": "gzip, deflate, br",
			"accept-language": "ja-JP;q=0.8,ja;q=0.7,en-US;q=0.6,en;q=0.5",
		}),
		RateLimit(2),
		Retry(5),
	)
	K := NewKemono(
		WithUsers("74671556", "fanbox"),
		WithBanner(true),
		WithPostFilter(
			ReleaseDateFilter(time.Now().AddDate(0, 0, -365), time.Now()),
		),
		WithAttachmentFilter(
			ExtensionFilter(".jpg", ".png", ".zip", ".gif"),
		),
		SetDownloader(downloader),
	)
	K.Start()
}
