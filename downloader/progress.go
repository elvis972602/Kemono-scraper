package downloader

import (
	"fmt"
	"github.com/elvis972602/kemono-scraper/utils"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DeepRed    = "\x1b[38;5;196m"
	Red        = "\x1b[38;5;197m"
	Green      = "\x1b[38;5;106m"
	DeepYellow = "\x1b[38;5;178m"
	Blue       = "\x1b[38;5;67m"
	Purple     = "\x1b[38;5;133m"
	Grey       = "\x1b[38;5;243m"
	White      = "\x1b[38;5;251m"
)

const (
	BarModeDownload = "Download"
	BarModeCancel   = "Cancel"
	BarModeFailed   = "Failed"
	BarModeSuccess  = "Success"
)

type progressBar struct {
	Start   time.Time
	Content string
	Max     int64
	cur     int64
	Length  int
	done    bool
}

func NewProgressBar(content string, max int64, length int) *progressBar {
	return &progressBar{Start: time.Now(), Content: content, Max: max, Length: length}
}

func (p *progressBar) Add(n int) {
	p.Add64(int64(n))
}

func (p *progressBar) Add64(n int64) {
	atomic.AddInt64(&p.cur, n)
}

func (p *progressBar) Set(n int) {
	p.Set64(int64(n))
}

func (p *progressBar) Set64(n int64) {
	atomic.StoreInt64(&p.cur, n)
}

func (p *progressBar) String(mode string) string {
	//var process string
	var pre float64
	if p.Max == 0 {
		pre = 0

	} else {
		pre = float64(p.cur) / float64(p.Max)
	}
	speed := int64(float64(p.cur) / time.Since(p.Start).Seconds())
	if speed < 0 {
		speed = 0
	}
	return buildProgressBar(utils.FormatDuration(int64(time.Since(p.Start))), mode, utils.FormatSize(speed), utils.FormatSize(p.Max), p.Content, pre, 30, mode)
}

func (p *progressBar) Done() {
	p.done = true
}

func (p *progressBar) IsDone() bool {
	return p.done
}

func (p *progressBar) Write(b []byte) (n int, err error) {
	n = len(b)
	p.Add(n)
	return
}

type Progress struct {
	progressBars []*progressBar
	count        int
	pre          int
	lock         sync.Mutex
	log          Log
}

func NewProgress(log Log) *Progress {
	return &Progress{pre: 0, log: log}
}

func (p *Progress) AddBar(bar *progressBar) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.progressBars = append(p.progressBars, bar)
}

func (p *Progress) Remove(bar *progressBar) {
	p.lock.Lock()
	defer p.lock.Unlock()
	for i, v := range p.progressBars {
		if v == bar {
			if i == len(p.progressBars)-1 {
				p.progressBars = p.progressBars[:i]
			} else {
				p.progressBars = append(p.progressBars[:i], p.progressBars[i+1:]...)
			}
		}
	}
}

func (p *Progress) Success(bar *progressBar) {
	bar.Done()
	p.Remove(bar)
	p.SetStatus()
	p.Print(bar.String(BarModeSuccess))
}

func (p *Progress) Failed(bar *progressBar, err error) {
	bar.Done()
	p.Remove(bar)
	p.SetStatus()
	p.Print(bar.String(BarModeFailed))
	p.Print(DeepRed + err.Error())
}

func (p *Progress) Cancel(bar *progressBar, err string) {
	bar.Done()
	p.Remove(bar)
	p.SetStatus()
	p.Print(bar.String(BarModeCancel))
	p.Print(DeepRed + err)
}

func (p *Progress) SetStatus() {
	var s []string
	p.lock.Lock()
	defer p.lock.Unlock()
	for i := 0; i < len(p.progressBars); i++ {
		s = append(s, p.progressBars[i].String(BarModeDownload))
	}
	if len(s) == 0 {
		s = append(s, "")
	}
	p.log.SetStatus(s)
}

func (p *Progress) Print(s string) {
	p.log.Print(s)
}

func (p *Progress) Run(interval time.Duration) {
	go func() {
		tick := time.NewTicker(interval)
		for {
			select {
			case <-tick.C:
				p.SetStatus()
			}
		}
	}()
}

func buildProgressBar(timeStr, prefix, speed, sizeStr, filename string, percent float64, length int, mode string) string {
	var barColor string
	switch mode {
	case BarModeDownload:
		barColor = Red
	case BarModeCancel:
		barColor = Grey
	case BarModeFailed:
		barColor = DeepRed
	case BarModeSuccess:
		barColor = Green
	}
	var process strings.Builder
	process.WriteString(DeepYellow)
	process.WriteString(fmt.Sprintf("%9s", timeStr))
	process.WriteString(" ")
	process.WriteString(White)
	process.WriteString(fmt.Sprintf("%8s", prefix))
	process.WriteString(" ")
	completedChars := int(percent * float64(length))
	process.WriteString(barColor)
	for i := 0; i < completedChars; i++ {
		process.WriteString("━")
	}
	if mode == BarModeDownload {
		process.WriteString(Grey)
		if completedChars > 0 && completedChars < length {
			completedChars++
			process.WriteString("╺")
		}
	}
	for i := completedChars; i < length; i++ {
		process.WriteString("━")
	}
	process.WriteString(" ")
	process.WriteString(Purple)
	process.WriteString(fmt.Sprintf("%5.1f", percent*100))
	process.WriteString("%")
	process.WriteString(" ")
	process.WriteString(Blue)
	process.WriteString(fmt.Sprintf("%10s", speed))
	process.WriteString("/s")
	process.WriteString(" ")
	process.WriteString(White)
	process.WriteString(fmt.Sprintf("%9s", sizeStr))
	process.WriteString(" ")
	process.WriteString(Grey)
	process.WriteString(filename)
	return process.String()
}
