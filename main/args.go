package main

import (
	"flag"
	"fmt"
	"github.com/mattn/go-colorable"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

var (
	help bool
	// url link
	link string
	// site
	site string
	// creator
	creator string
	// download banner
	banner bool

	// post filter
	// overwrite
	overwrite bool
	// first n posts
	first int
	// last n posts
	last int
	// date
	date int
	//date before
	dateBefore int
	//date after
	dateAfter int
	// update
	update int
	//update before
	updateBefore int
	//update after
	updateAfter int
	// extension only
	extensionOnly string
	// extension exclude
	extensionExclude string

	// download options
	// output directory
	output string
	// path template
	template string
	// Image template
	imageTemplate string
	// video template
	videoTemplate string
	// audio template
	audioTemplate string
	// archive template
	archiveTemplate string
	// content
	content bool
	// async
	async bool
	// max size
	maxSize string
	// min size
	minSize string
	// with prefix number
	withPrefixNumber bool
	// name rule only index
	nameRuleOnlyIndex bool
	// download timeout
	downloadTimeout int
	// download retry
	retry int
	// download retry interval
	retryInterval float64
	// max download goroutine
	maxDownloadParallel int
	// request per second
	rateLimit int
	// proxy url
	proxy string

	// download favorite creator
	favoriteCreator bool
	// download favorite post
	favoritePost bool
	// cookie browser
	cookieBrowser string
	// cookie file
	cookieFile string
)

var (
	config      map[string]interface{}
	passedFlags = make(map[string]bool)
)

func init() {
	log.SetOutput(colorable.NewColorableStdout())

	flag.BoolVar(&help, "help", false, "show all usage")
	flag.StringVar(&link, "link", "", "download link, should be same site, separate by comma")
	// if already have link, or creator, site will be ignored
	flag.StringVar(&site, "fav-site", "", "download favorite creator or post, separate by comma")
	flag.StringVar(&creator, "creator", "", "--creator <service>:<id>, separate by comma")
	flag.BoolVar(&banner, "banner", false, "if download banner")
	flag.BoolVar(&favoriteCreator, "fav-creator", false, "download favorite creator")
	flag.BoolVar(&favoritePost, "fav-post", false, "download favorite post")
	flag.StringVar(&cookieBrowser, "cookie-browser", "chrome", "cookie browser, windows only, support chrome, firefox, opera, edge, vivaldi,default is chrome, other system can use --cookie <file name>")
	flag.StringVar(&cookieFile, "cookie", "cookies.txt", "cookie file, default is cookies.txt (value separate by whitespace)\n"+
		"structure : +--------+--------------------+------+--------+--------+------+-------+\n"+
		"            | Domain | Include subdomains | Path | Secure | Expiry | Name | Value |\n"+
		"            +--------+--------------------+------+--------+--------+------+-------+")
	// filter
	flag.BoolVar(&overwrite, "overwrite", false, "if overwrite file")
	flag.IntVar(&first, "first", 0, "download first n posts")
	flag.IntVar(&last, "last", 0, "download last n posts")
	flag.IntVar(&date, "date", 0, "--date YYYYMMDD (notice: date in website is GMT+0)")
	flag.IntVar(&dateBefore, "date-before", 0, "--date-before YYYYMMDD, select posts before YYYYMMDD")
	flag.IntVar(&dateAfter, "date-after", 0, "--date-after YYYYMMDD, select posts after YYYYMMDD")
	flag.IntVar(&update, "update", 0, "--update YYYYMMDD (notice: date in website is GMT+0)")
	flag.IntVar(&updateBefore, "update-before", 0, "--update-before YYYYMMDD, select posts updated before YYYYMMDD")
	flag.IntVar(&updateAfter, "update-after", 0, "--update-after YYYYMMDD, select posts updated after YYYYMMDD")
	flag.StringVar(&extensionOnly, "extension-only", "", "--extension-only, select posts with only extension, separate by comma (e.g. --extension-only jpg,png)")
	flag.StringVar(&extensionExclude, "extension-exclude", "", "--extension-exclude, select posts without extension, separate by comma (e.g. --extension-exclude jpg,png)")

	// download options
	flag.StringVar(&output, "output", "", "output directory")
	flag.StringVar(&template, "template", "", "default path template, e.g. <ks:creator>/<ks:post>/<ks:index>_<ks:filename><ks:extension>")
	flag.StringVar(&imageTemplate, "image-template", "", "image template, e.g. <ks:creator>/<ks:post>/<ks:index><ks:extension>")
	flag.StringVar(&videoTemplate, "video-template", "", "video template, e.g. <ks:creator>/<ks:post>/<ks:filename><ks:extension>")
	flag.StringVar(&audioTemplate, "audio-template", "", "audio template, e.g. <ks:creator>/<ks:post>/<ks:filename><ks:extension>")
	flag.StringVar(&archiveTemplate, "archive-template", "", "archive template, e.g. <ks:creator>/<ks:post>/<ks:filename><ks:extension>")
	flag.BoolVar(&content, "content", false, "if download post content")
	flag.BoolVar(&async, "async", false, "if download posts asynchronously, may cause the file order is not the same as the post order, can be used with --with-prefix-number, default false")
	flag.StringVar(&maxSize, "max-size", "", "max size, e.g. 10 MB, 1 GB")
	flag.StringVar(&minSize, "min-size", "", "min size, e.g. 10 MB, 1 GB")
	flag.BoolVar(&withPrefixNumber, "with-prefix-number", false, "if add prefix number to file name: <index>-<file name> (zip file name is not changed)")
	flag.BoolVar(&nameRuleOnlyIndex, "name-rule-only-index", false, "if use only index as file name(eg. 1.png, 2.png, ...)")
	flag.IntVar(&downloadTimeout, "download-timeout", 1800, "download timeout(second), default is 1800s")
	flag.IntVar(&retry, "retry", 3, "download retry, default is 3")
	flag.Float64Var(&retryInterval, "retry-interval", 10, "download retry interval(second), default is 10s")
	flag.IntVar(&maxDownloadParallel, "max-download-parallel", 3, "max download file concurrent, default is 3, async mode only")
	flag.IntVar(&rateLimit, "rate-limit", 2, "request per second, default is 2")
	flag.StringVar(&proxy, "proxy", "", "proxy url, e.g. http://proxy.com:8080")
	_, err := os.Stat("config.yaml")

	if os.IsNotExist(err) {
		// file does not exist
		return
	}
	if err != nil {
		log.Printf("check config.yaml failed, %v", err)
		return
	}
	open, err := os.Open("config.yaml")
	if err != nil {
		log.Fatalf("open config file error: %v", err)
	}
	defer open.Close()
	bytes, err := ioutil.ReadAll(open)
	if err != nil {
		log.Fatalf("read config file error: %v", err)
	}
	err = yaml.Unmarshal(bytes, &config)
	if err != nil {
		log.Fatalf("unmarshal config file error: %v", err)
	}
}

func setPassedFlags() {
	flag.Visit(func(f *flag.Flag) {
		passedFlags[f.Name] = true
	})
}

// PrintDefaults same as flag.PrintDefaults(), but only print the flag with two hyphens and without default value
func PrintDefaults() {
	flag.VisitAll(func(f *flag.Flag) {
		var b strings.Builder
		fmt.Fprintf(&b, "  --%s", f.Name) // Two spaces before -; see next two comments.
		name, usage := flag.UnquoteUsage(f)
		if len(name) > 0 {
			b.WriteString(" ")
			b.WriteString(name)
		}
		// Boolean flags of one ASCII letter are so common we
		// treat them specially, putting their usage on the same line.
		if b.Len() <= 4 { // space, space, '-', 'x'.
			b.WriteString("\t")
		} else {
			// Four spaces before the tab triggers good alignment
			// for both 4- and 8-space tab stops.
			b.WriteString("\n    \t")
		}
		b.WriteString(strings.ReplaceAll(usage, "\n", "\n    \t"))
		b.WriteString("\n")
		fmt.Fprint(flag.CommandLine.Output(), b.String())
	})
}
