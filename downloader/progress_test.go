package downloader

import "testing"

func TestProgressBar_Builder(t *testing.T) {
	// download
	bar := NewProgressBar(
		"test",
		100,
		30,
	)
	for i := 0; i <= 100; i++ {
		t.Log(bar.String(BarModeDownload))
		bar.Add(1)
	}

	// success
	bar = NewProgressBar(
		"test",
		100,
		30,
	)
	for i := 0; i <= 100; i++ {
		t.Log(bar.String(BarModeSuccess))
		bar.Add(1)
	}

	// failed
	bar = NewProgressBar(
		"test",
		100,
		30,
	)
	for i := 0; i <= 100; i++ {
		t.Log(bar.String(BarModeFailed))
		bar.Add(1)
	}

	// cancel
	bar = NewProgressBar(
		"test",
		100,
		30,
	)
	for i := 0; i <= 100; i++ {
		t.Log(bar.String(BarModeCancel))
		bar.Add(1)
	}

}
