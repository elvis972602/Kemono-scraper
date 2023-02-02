package kemono_scraper

import (
	"crypto/sha256"
	"encoding/json"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func SplitHash(str string) (string, error) {
	parts := strings.Split(str, "/")
	if len(parts) < 4 {
		return "", nil
	}
	ext := filepath.Ext(parts[3])
	name := parts[3][:len(parts[3])-len(ext)]
	return name, nil
}

// Stringify beautify interface to string
func Stringify(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

func Hash(w io.Reader) ([]byte, error) {
	ha := sha256.New()
	_, err := io.Copy(ha, w)
	if err != nil {
		return nil, err
	}
	return ha.Sum(nil), nil
}

func ValidDirectoryName(name string) string {
	if runtime.GOOS == "windows" {
		invalidRune := "/\\:*?\"<>|"
		validate := func(r rune) rune {
			if strings.ContainsRune(invalidRune, r) {
				return '_'
			}
			return r
		}
		if strings.ContainsAny(name, `/\:*?"<>|`) {
			return strings.TrimSpace(strings.Map(validate, name))
		} else {
			return name
		}
	}
	invalidRune := "/\\"
	validate := func(r rune) rune {
		if strings.ContainsRune(invalidRune, r) {
			return '_'
		}
		return r
	}
	if strings.ContainsAny(name, `/\`) {
		return strings.TrimSpace(strings.Map(validate, name))
	} else {
		return name
	}
}

type semaphore chan struct{}

func newSemaphore(n int) semaphore {
	return make(semaphore, n)
}

func (s semaphore) acquire() {
	s <- struct{}{}
}

func (s semaphore) release() {
	<-s
}

type rateLimiter struct {
	limit     int
	semaphore semaphore
}

func newRateLimiter(tokenPreSecond int) *rateLimiter {
	r := &rateLimiter{
		limit:     tokenPreSecond,
		semaphore: newSemaphore(tokenPreSecond),
	}
	r.Timing()
	return r
}

// Timing add token into semaphore
func (r *rateLimiter) Timing() {
	t := time.NewTicker(time.Second)
	// full semaphore
	for i := 0; i < r.limit; i++ {
		r.semaphore.acquire()
	}
	go func() {
		for {
			select {
			case <-t.C:
				for i := 0; i < r.limit; i++ {
					r.semaphore.acquire()
				}
			}
		}
	}()
}

func (r *rateLimiter) Token() {
	r.semaphore.release()
}
