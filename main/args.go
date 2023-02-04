package main

import (
	"flag"
	"fmt"
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
	// data
	data string
	//data before
	dataBefore string
	//data after
	dataAfter string
	// update
	update string
	//update before
	updateBefore string
	//update after
	updateAfter string
	// extension only
	extensionOnly string
	// extension exclude
	extensionExclude string

	// download options
	// output directory
	output string
	// async
	async bool
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
)

var config map[string]interface{}

func init() {
	flag.BoolVar(&help, "help", false, "show all usage")
	flag.StringVar(&link, "link", "", "download link, should be same site, separate by comma")
	flag.StringVar(&site, "site", "", "download site, should be same as link")
	flag.StringVar(&creator, "creator", "", "--creator <service>:<id>, separate by comma")
	flag.BoolVar(&banner, "banner", false, "if download banner")

	// filter
	flag.BoolVar(&overwrite, "overwrite", false, "if overwrite file")
	flag.IntVar(&first, "first", 0, "download first n posts")
	flag.IntVar(&last, "last", 0, "download last n posts")
	flag.StringVar(&data, "data", "", "--data YYYYMMDD (notice: data in website is GMT+0)")
	flag.StringVar(&dataBefore, "data-before", "", "--data-before YYYYMMDD, select posts before YYYYMMDD")
	flag.StringVar(&dataAfter, "data-after", "", "--data-after YYYYMMDD, select posts after YYYYMMDD")
	flag.StringVar(&update, "update", "", "--update YYYYMMDD (notice: data in website is GMT+0)")
	flag.StringVar(&updateBefore, "update-before", "", "--update-before YYYYMMDD, select posts updated before YYYYMMDD")
	flag.StringVar(&updateAfter, "update-after", "", "--update-after YYYYMMDD, select posts updated after YYYYMMDD")
	flag.StringVar(&extensionOnly, "extension-only", "", "--extension-only, select posts with only extension, separate by comma (e.g. --extension-only jpg,png)")
	flag.StringVar(&extensionExclude, "extension-exclude", "", "--extension-exclude, select posts without extension, separate by comma (e.g. --extension-exclude jpg,png)")

	// download options
	flag.StringVar(&output, "output", "", "output directory")
	flag.BoolVar(&async, "async", false, "if download posts asynchronously, may cause the file order is not the same as the post order, can be used with --with-prefix-number, default false")
	flag.BoolVar(&withPrefixNumber, "with-prefix-number", false, "if add prefix number to file name: <index>-<file name> (zip file name is not changed)")
	flag.BoolVar(&nameRuleOnlyIndex, "name-rule-only-index", false, "if use only index as file name(eg. 1.png, 2.png, ...)")
	flag.IntVar(&downloadTimeout, "download-timeout", 300, "download timeout(second), default is 300s")
	flag.IntVar(&retry, "retry", 3, "download retry, default is 3")
	flag.Float64Var(&retryInterval, "retry-interval", 10, "download retry interval(second), default is 10s")
	flag.IntVar(&maxDownloadParallel, "max-download-parallel", 3, "max download file concurrent, default is 3, async mode only")
	flag.IntVar(&rateLimit, "rate-limit", 2, "request per second, default is 2")

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

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
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
