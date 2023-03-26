package utils

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	PreLine   = "\033[1A\r"
	ClearLine = "\033[2K\r"
	SeekFirst = "\r"
)

type Log interface {
	Printf(format string, v ...interface{})
	Print(s string)
	SetStatus(s []string)
}

type Bar struct {
	Since   time.Time
	Prefix  string
	Content string
	Max     int64
	cur     int64
	Length  int
	done    bool
}

func (p *Bar) Add(n int64) {
	p.cur += n
}

func (p *Bar) Set(n int64) {
	p.cur = n
}

func (p *Bar) String() string {
	var process string
	var pre float64
	if p.Max == 0 {
		pre = 0

	} else {
		pre = float64(p.cur) / float64(p.Max) * 100
	}
	intPre := int(pre * float64(p.Length) / 100)
	speed := int64(float64(p.cur) / time.Since(p.Since).Seconds())
	if speed < 0 {
		speed = 0
	}
	if p.cur == 0 {
		process = processString("fetching...", p.Length+1)
	} else {
		process = string(bytes.Repeat([]byte{'='}, intPre)) + ">" + string(bytes.Repeat([]byte{' '}, p.Length-intPre))
	}
	return ShortenString(fmt.Sprintf("[%08s]%s[%s] %.1f%% (%s/s) [%s]", FormatDuration(int64(time.Since(p.Since))), p.Prefix, process, pre, FormatSize(speed), FormatSize(p.Max)), fmt.Sprintf("%s", p.Content), "")
}

func (p *Bar) FailString(err string) string {
	var pre float64
	if p.Max == 0 {
		pre = 0

	} else {
		pre = float64(p.cur) / float64(p.Max) * 100
	}
	intPre := int(pre * float64(p.Length) / 100)
	speed := int64(float64(p.cur) / time.Since(p.Since).Seconds())
	if speed < 0 {
		speed = 0
	}
	return ShortenString(fmt.Sprintf("\033[31m[Failed]\033[0m%s[%s] %.1f%% (%s/s) [%s] ", err, strings.Repeat("=", intPre)+">"+strings.Repeat("x", p.Length-intPre), pre, FormatSize(speed), FormatSize(p.Max)), fmt.Sprintf("%s", p.Content), "")
}

func (p *Bar) CancelString(err string) string {
	var process string
	var pre float64
	if p.Max == 0 {
		pre = 0
	} else {
		pre = float64(p.cur) / float64(p.Max) * 100
	}
	speed := int64(float64(p.cur) / time.Since(p.Since).Seconds())
	if speed < 0 {
		speed = 0
	}
	if len(err) > p.Length {
		err = err[:p.Length]
	}
	process = processString(err, p.Length+1)
	return ShortenString(fmt.Sprintf("[%08s]%s[%s] %.1f%% (%s/s) [%s]", FormatDuration(int64(time.Since(p.Since))), p.Prefix, process, pre, FormatSize(speed), FormatSize(p.Max)), fmt.Sprintf("%s", p.Content), "")
}

func processString(s string, length int) string {
	if len(s) > length {
		return s[:length]
	}
	return string(bytes.Repeat([]byte{' '}, (length-len(s))/2)) + s + string(bytes.Repeat([]byte{' '}, length-len(s)-((length-len(s))/2)))
}

func (p *Bar) Done() {
	p.done = true
}

func (p *Bar) IsDone() bool {
	return p.done
}

type ProgressBar struct {
	progressBars []*Bar
	count        int
	pre          int
	lock         sync.Mutex
	log          Log
}

func NewProgressBar(log Log) *ProgressBar {
	return &ProgressBar{pre: 0, log: log}
}

func (p *ProgressBar) AddBar(bar *Bar) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.progressBars = append(p.progressBars, bar)
}

func (p *ProgressBar) Remove(bar *Bar) {
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

func (p *ProgressBar) Success(bar *Bar) {
	bar.Done()
	p.Remove(bar)
	p.SetStatus()
	p.Print(bar.String())
}

func (p *ProgressBar) Failed(bar *Bar, err error) {
	bar.Done()
	p.Remove(bar)
	p.SetStatus()
	p.Print(bar.FailString(err.Error()))
}

func (p *ProgressBar) Cancel(bar *Bar, err string) {
	bar.Done()
	p.Remove(bar)
	p.SetStatus()
	p.Print(bar.CancelString(err))
}

func (p *ProgressBar) SetStatus() {
	var s []string
	p.lock.Lock()
	defer p.lock.Unlock()
	for i := 0; i < len(p.progressBars); i++ {
		s = append(s, p.progressBars[i].String())
	}
	if len(s) == 0 {
		s = append(s, "")
	}
	p.log.SetStatus(s)
}

func (p *ProgressBar) Print(s string) {
	p.log.Print(s)
}
