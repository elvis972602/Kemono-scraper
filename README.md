# Kemono-scraper
A simple scraper to  filter and download images from kemono.party

## Usage
```go
go get github.com/elvis972602/kemono-scraper
```

## Example
```go
K := NewKemono(
	WithUsers("123456", "service", "654321", "service2"),
	WithPostFilter(
		ReleaseDateFilter(time.Now().AddDate(0, 0, -365), time.Now()),
	),
	WithAttachmentFilter(
		ExtensionFilter(".jpg", ".png", ".zip"),
	),
        SetDownloader(NewDownloader(
        	Async(true),
        ),
        ), 
)
K.Start()
```

## Features
With Kemono-scraper, you can implement a Downloader to take advantage of features such as multi-connection downloading, resume broken downloads, and more.




