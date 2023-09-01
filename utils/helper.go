package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	if name == "" {
		return ""
	}
	if runtime.GOOS == "windows" {
		invalidRune := "\x00-\x1f/\\:*?\"<>|\n\r\t"
		validate := func(r rune) rune {
			if strings.ContainsRune(invalidRune, r) {
				return '_'
			}
			return r
		}
		s := strings.TrimSpace(strings.Map(validate, name))
		if len(s) > 200 {
			s = s[:200]
		}
		if len(s) > 0 && s[len(s)-1] == '.' {
			s = s[:len(s)-1]
			return fmt.Sprintf("%s_", s)
		}
		return s
	}
	invalidRune := "/\\\n\r\t"
	validate := func(r rune) rune {
		if strings.ContainsRune(invalidRune, r) {
			return '_'
		}
		return r
	}
	s := strings.TrimSpace(strings.Map(validate, name))
	if len(s) > 200 {
		s = s[:200]
	}
	if len(s) > 0 && s[0] == '.' {
		s = s[1:]
		return fmt.Sprintf("_%s", s)
	}
	return s
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

type RateLimiter struct {
	limit     int
	semaphore semaphore
}

func NewRateLimiter(tokenPreSecond int) *RateLimiter {
	r := &RateLimiter{
		limit:     tokenPreSecond,
		semaphore: newSemaphore(tokenPreSecond),
	}
	r.Timing()
	return r
}

// Timing add token into semaphore
func (r *RateLimiter) Timing() {
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

func (r *RateLimiter) Token() {
	r.semaphore.release()
}

func GenerateToken(size int) (string, error) {
	data := make([]byte, size)
	_, err := rand.Read(data)
	if err != nil {
		return "", err
	}

	// Convert to hexadecimal
	hexStr := hex.EncodeToString(data)

	return hexStr, nil

}
