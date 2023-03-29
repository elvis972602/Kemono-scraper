package downloader

import (
	"fmt"
	"github.com/elvis972602/kemono-scraper/utils"
	"strings"
	"sync"
	"time"
)

const (
	DeepBlack  = "\x1b[38;5;235m"
	Black      = "\x1b[38;5;238m"
	DeepRed    = "\x1b[38;5;196m"
	Red        = "\x1b[38;5;197m"
	DeepGreen  = "\x1b[38;5;76m"
	Green      = "\x1b[38;5;76m"
	DeepYellow = "\x1b[38;5;214m"
	Yellow     = "\x1b[38;5;226m"
	DeepBlue   = "\x1b[38;5;21m"
	Blue       = "\x1b[38;5;38m"
	DeepPurple = "\x1b[38;5;141m"
	Purple     = "\x1b[38;5;134m"
	DeepCyan   = "\x1b[38;5;37m"
	Cyan       = "\x1b[38;5;39m"
	Grey       = "\x1b[38;5;242m"
	White      = "\x1b[38;5;255m"
	DeepWhite  = "\x1b[38;5;254m"
)

const (
	BarModeDownload = "Download"
	BarModeCancel   = "Cancel"
	BarModeFailed   = "Failed"
	BarModeSuccess  = "Success"
)

type ProgressBar struct {
	Start   time.Time
	Content string
	Max     int64
	cur     int64
	Length  int
	done    bool
}

func NewProgressBar(content string, max int64, length int) *ProgressBar {
	return &ProgressBar{Start: time.Now(), Content: content, Max: max, Length: length}
}

func (p *ProgressBar) Add(n int64) {
	p.cur += n
}

func (p *ProgressBar) Set(n int64) {
	p.cur = n
}

func (p *ProgressBar) String(mode string) string {
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

func (p *ProgressBar) Done() {
	p.done = true
}

func (p *ProgressBar) IsDone() bool {
	return p.done
}

type Progress struct {
	progressBars []*ProgressBar
	count        int
	pre          int
	lock         sync.Mutex
	log          Log
}

func NewProgress(log Log) *Progress {
	return &Progress{pre: 0, log: log}
}

func (p *Progress) AddBar(bar *ProgressBar) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.progressBars = append(p.progressBars, bar)
}

func (p *Progress) Remove(bar *ProgressBar) {
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

func (p *Progress) Success(bar *ProgressBar) {
	bar.Done()
	p.Remove(bar)
	p.SetStatus()
	p.Print(bar.String(BarModeSuccess))
}

func (p *Progress) Failed(bar *ProgressBar, err error) {
	bar.Done()
	p.Remove(bar)
	p.SetStatus()
	p.Print(bar.String(BarModeFailed))
	p.Print(DeepRed + err.Error())
}

func (p *Progress) Cancel(bar *ProgressBar, err string) {
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
	process.WriteString(fmt.Sprintf("%9s", speed))
	process.WriteString("/s")
	process.WriteString(" ")
	process.WriteString(White)
	process.WriteString(fmt.Sprintf("%9s", sizeStr))
	process.WriteString(" ")
	process.WriteString(Grey)
	process.WriteString(filename)
	return process.String()
}
