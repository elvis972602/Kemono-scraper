package main

import (
	"time"
)

func main() {
	K := NewKemono(
		WithUsers("70050825", "fanbox"),
		WithPostFilter(
			ReleaseDateFilter(time.Now().AddDate(0, 0, -365), time.Now()),
		),
		WithAttachmentFilter(
			ExtensionFilter(".jpg", ".png", ".zip"),
		),
	)
	K.Start()
}
