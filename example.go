package main

import (
	"time"
)

func main() {
	K := NewKemono(
		WithUsers("123456", "service", "654321", "service2"),
		WithPostFilter(
			ReleaseDateFilter(time.Now().AddDate(0, 0, -365), time.Now()),
		),
		WithAttachmentFilter(
			ExtensionFilter(".jpg", ".png", ".zip"),
		),
	)
	K.Start()
}
