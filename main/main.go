package main

import (
	"flag"
	"fmt"
	kemono "github.com/elvis972602/kemono-scraper"
	"log"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func main() {
	flag.Parse()

	if help {
		PrintDefaults()
		return
	}

	setFlag()

	var (
		creators               []string
		links                  []string
		options                []kemono.Option
		downloaderOptions      []kemono.DownloadOption
		hasLink                bool
		s, srv, userId, postId string
	)

	if creator != "" {
		creatorComponents := strings.Split(creator, ",")
		for _, c := range creatorComponents {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			creators = append(creators, c)
		}
	}

	if link != "" {
		linkComponents := strings.Split(link, ",")
		for _, l := range linkComponents {
			l = strings.TrimSpace(l)
			if l == "" {
				continue
			}
			links = append(links, l)
		}
	}
	downloaderOptions = append(downloaderOptions, kemono.Async(async))

	if len(links) > 0 {
		hasLink = true
		users := make([]string, 0)
		ids := make([]string, 0)
		ss := ""
		for i, l := range links {
			s, srv, userId, postId = parasLink(l)
			if i == 0 {
				ss = s
			} else {
				if s != ss {
					log.Fatalf("site %s is not match %s", s, ss)
				}
			}
			users = append(users, userId, srv)
			ids = append(ids, postId)
		}
		options = append(options,
			kemono.WithDomain(s),
			kemono.WithUsers(users...),
			kemono.WithPostFilter(
				kemono.IdFilter(ids...),
			),
		)
		downloaderOptions = append(downloaderOptions, kemono.BaseURL(fmt.Sprintf("https://%s.party", s)))
	}

	// check site
	if site != "" {
		if hasLink {
			if site != s {
				log.Fatalf("site %s not match link %s", site, link)
			}
		} else {
			options = append(options, kemono.WithDomain(site))
			downloaderOptions = append(downloaderOptions, kemono.BaseURL(fmt.Sprintf("https://%s.party", site)))
		}
	} else {
		if !hasLink {
			site = "kemono"
			options = append(options, kemono.WithDomain(site))
			downloaderOptions = append(downloaderOptions, kemono.BaseURL(fmt.Sprintf("https://%s.party", site)))

		}
	}

	// check creator
	if len(creators) > 0 {
		users := make([]string, 0)
		for _, c := range creators {
			creatorComponents := strings.Split(c, ":")
			if len(creatorComponents) != 2 {
				log.Fatalf("invalid creator %s", c)
			}
			users = append(users, creatorComponents[1], creatorComponents[0])
		}
		options = append(options, kemono.WithUsers(users...))
	}

	// banner
	options = append(options, kemono.WithBanner(banner))

	// overwrite
	downloaderOptions = append(downloaderOptions, kemono.OverWrite(overwrite))

	// check first
	if first != 0 {
		options = append(options, kemono.WithPostFilter(
			kemono.NumbFilter(func(i int) bool {
				if i <= first {
					return true
				}
				return false
			}),
		))
	}

	// check last
	if last != 0 {
		options = append(options, kemono.WithPostFilter(
			kemono.NumbFilter(func(i int) bool {
				if i >= last {
					return true
				}
				return false
			}),
		))
	}

	if data != "" {
		t := parasData(data)
		options = append(options, kemono.WithPostFilter(
			kemono.ReleaseDateFilter(t.Add(-1), t.Add(24*time.Hour-1)),
		))
	}

	if dataBefore != "" {
		t := parasData(dataBefore)
		options = append(options, kemono.WithPostFilter(
			kemono.ReleaseDateBeforeFilter(t),
		))
	}

	if dataAfter != "" {
		t := parasData(dataAfter)
		options = append(options, kemono.WithPostFilter(
			kemono.ReleaseDateAfterFilter(t),
		))
	}

	if update != "" {
		t := parasData(update)
		options = append(options, kemono.WithPostFilter(
			kemono.EditDateFilter(t.Add(-1), t.Add(24*time.Hour-1)),
		))
	}

	if updateBefore != "" {
		t := parasData(updateBefore)
		options = append(options, kemono.WithPostFilter(
			kemono.EditDateBeforeFilter(t),
		))
	}

	if updateAfter != "" {
		t := parasData(updateAfter)
		options = append(options, kemono.WithPostFilter(
			kemono.EditDateAfterFilter(t),
		))
	}

	// check extensionOnly
	if extensionOnly != "" {
		extensionComponents := strings.Split(extensionOnly, ",")
		// check extension has dot
		for i, extension := range extensionComponents {
			extension = strings.TrimSpace(extension)
			if !strings.HasPrefix(extension, ".") {
				extensionComponents[i] = "." + extension
			}
		}
		options = append(options, kemono.WithAttachmentFilter(
			kemono.ExtensionFilter(extensionComponents...),
		))
	}

	// check extensionExclude
	if extensionExclude != "" {
		extensionComponents := strings.Split(extensionExclude, ",")
		// check extension has dot
		for i, extension := range extensionComponents {
			extension = strings.TrimSpace(extension)
			if !strings.HasPrefix(extension, ".") {
				extensionComponents[i] = "." + extension
			}
		}
		options = append(options, kemono.WithAttachmentFilter(
			kemono.ExtensionExcludeFilter(extensionComponents...),
		))
	}

	// check output
	if output != "" {
		var pathFunc func(creator kemono.Creator, post kemono.Post, i int, attachment kemono.File) string
		ruleflag := false
		if nameRuleOnlyIndex {
			ruleflag = true
			pathFunc = func(creator kemono.Creator, post kemono.Post, i int, attachment kemono.File) string {
				ext := filepath.Ext(attachment.Name)
				name := fmt.Sprintf("%d%s", i, ext)
				return fmt.Sprintf(filepath.Join("%s", "%s", "%s", "%s"), output, kemono.ValidDirectoryName(creator.Name), kemono.ValidDirectoryName(post.Title), kemono.ValidDirectoryName(name))
			}
		}
		if withPrefixNumber {
			ruleflag = true
			pathFunc = func(creator kemono.Creator, post kemono.Post, i int, attachment kemono.File) string {
				var name string
				if filepath.Ext(attachment.Name) == ".zip" {
					name = attachment.Name
				} else {
					name = fmt.Sprintf("%d-%s", i, attachment.Name)
				}
				return fmt.Sprintf(filepath.Join("%s", "%s", "%s", "%s"), output, kemono.ValidDirectoryName(creator.Name), kemono.ValidDirectoryName(post.Title), kemono.ValidDirectoryName(name))
			}
		}
		if !ruleflag {
			pathFunc = func(creator kemono.Creator, post kemono.Post, i int, attachment kemono.File) string {
				return fmt.Sprintf(filepath.Join("%s", "%s", "%s", "%s"), output, kemono.ValidDirectoryName(creator.Name), kemono.ValidDirectoryName(post.Title), kemono.ValidDirectoryName(attachment.Name))
			}
		}
		downloaderOptions = append(downloaderOptions, kemono.SavePath(pathFunc))
	} else {
		if withPrefixNumber {
			var pathFunc func(creator kemono.Creator, post kemono.Post, i int, attachment kemono.File) string
			pathFunc = func(creator kemono.Creator, post kemono.Post, i int, attachment kemono.File) string {
				var name string
				if filepath.Ext(attachment.Name) == ".zip" {
					name = attachment.Name
				} else {
					name = fmt.Sprintf("%d-%s", i, attachment.Name)
				}
				return fmt.Sprintf(filepath.Join("./download", "%s", "%s", "%s"), kemono.ValidDirectoryName(creator.Name), kemono.ValidDirectoryName(post.Title), kemono.ValidDirectoryName(name))
			}
			downloaderOptions = append(downloaderOptions, kemono.SavePath(pathFunc))
		}

		if nameRuleOnlyIndex {
			var pathFunc func(creator kemono.Creator, post kemono.Post, i int, attachment kemono.File) string
			pathFunc = func(creator kemono.Creator, post kemono.Post, i int, attachment kemono.File) string {
				ext := filepath.Ext(attachment.Name)
				name := fmt.Sprintf("%d%s", i, ext)
				return fmt.Sprintf(filepath.Join("./download", "%s", "%s", "%s"), kemono.ValidDirectoryName(creator.Name), kemono.ValidDirectoryName(post.Title), kemono.ValidDirectoryName(name))
			}
			downloaderOptions = append(downloaderOptions, kemono.SavePath(pathFunc))
		}
	}

	if downloadTimeout <= 0 {
		log.Fatalf("invalid download timeout %d", downloadTimeout)
	} else {
		downloaderOptions = append(downloaderOptions, kemono.Timeout(time.Duration(downloadTimeout)*time.Second))
	}

	if retry < 0 {
		log.Fatalf("retry must be greater than 0")
	} else {
		downloaderOptions = append(downloaderOptions, kemono.Retry(retry))
	}

	if retryInterval < 0 {
		log.Fatalf("retry interval must be greater than 0")
	} else {
		downloaderOptions = append(downloaderOptions, kemono.RetryInterval(time.Duration(retryInterval)*time.Second))
	}

	// check maxDownloadGoroutine
	if maxDownloadParallel <= 0 {
		log.Fatalf("maxDownloadParallel must be greater than 0")
	} else {
		downloaderOptions = append(downloaderOptions, kemono.MaxConcurrent(maxDownloadParallel))
	}

	if rateLimit <= 0 {
		log.Fatalf("rate limit must be greater than 0")
	} else {
		downloaderOptions = append(downloaderOptions, kemono.RateLimit(rateLimit))
	}

	downloader := kemono.NewDownloader(downloaderOptions...)

	options = append(options, kemono.SetDownloader(downloader))

	K := kemono.NewKemono(options...)

	K.Start()

}

func parasLink(link string) (site, service, userId, postId string) {
	var ext string

	u, err := url.Parse(link)
	if err != nil {
		log.Fatal("invalid url")
	}

	hostComponents := strings.Split(u.Host, ".")
	if len(hostComponents) != 2 {
		log.Fatal("Error splitting host component:", u.Host)
		return
	}

	site = hostComponents[0]
	ext = hostComponents[1]
	if ext != "party" {
		log.Fatal("invalid url")
	}

	pathComponents := strings.Split(u.Path, "/")
	if len(pathComponents) != 6 && len(pathComponents) != 4 {
		log.Fatal("Error splitting host component:", pathComponents, len(pathComponents))
		return
	}
	if len(pathComponents) == 6 {
		service = pathComponents[1]
		userId = pathComponents[3]
		postId = pathComponents[5]
	} else {
		service = pathComponents[1]
		userId = pathComponents[3]
	}

	return
}

func parasData(data string) time.Time {
	if len(data) != 8 {
		log.Fatalf("invalid data %s", data)
	}
	year, err := strconv.Atoi(data[:4])
	if err != nil {
		log.Fatalf("invalid data %s", data)
	}
	month, err := strconv.Atoi(data[4:6])
	if err != nil {
		log.Fatalf("invalid data %s", data)
	}
	day, err := strconv.Atoi(data[6:])
	if err != nil {
		log.Fatalf("invalid data %s", data)
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func setFlag() {
	if !isFlagPassed("link") && config["link"] != nil {
		link = config["link"].(string)
	}
	if !isFlagPassed("site") && config["site"] != nil {
		site = config["site"].(string)
	}
	if !isFlagPassed("creator") && config["creator"] != nil {
		creator = config["creator"].(string)
	}
	if !isFlagPassed("banner") && config["banner"] != nil {
		banner = config["banner"].(bool)
	}
	if !isFlagPassed("overwrite") && config["overwrite"] != nil {
		overwrite = config["overwrite"].(bool)
	}
	if !isFlagPassed("first") && config["first"] != nil {
		first = config["first"].(int)
	}
	if !isFlagPassed("last") && config["last"] != nil {
		last = config["last"].(int)
	}
	if !isFlagPassed("data") && config["data"] != nil {
		data = config["data"].(string)
	}
	if !isFlagPassed("data-before") && config["data-before"] != nil {
		data = config["data-before"].(string)
	}
	if !isFlagPassed("data-after") && config["data-after"] != nil {
		data = config["data-after"].(string)
	}
	if !isFlagPassed("update") && config["update"] != nil {
		update = config["update"].(string)
	}
	if !isFlagPassed("update-before") && config["update-before"] != nil {
		update = config["update-before"].(string)
	}
	if !isFlagPassed("update-after") && config["update-after"] != nil {
		update = config["update-after"].(string)
	}
	if !isFlagPassed("extension-only") && config["extension-only"] != nil {
		extensionOnly = config["extension-only"].(string)
	}
	if !isFlagPassed("extension-exclude") && config["extension-exclude"] != nil {
		extensionExclude = config["extension-exclude"].(string)
	}
	if !isFlagPassed("output") && config["output"] != nil {
		output = config["output"].(string)
	}
	if !isFlagPassed("async") && config["async"] != nil {
		async = config["async"].(bool)
	}
	if !isFlagPassed("with-prefix-number") && config["with-prefix-number"] != nil {
		withPrefixNumber = config["with-prefix-number"].(bool)
	}
	if !isFlagPassed("name-rule-only-index") && config["name-rule-only-index"] != nil {
		nameRuleOnlyIndex = config["name-rule-only-index"].(bool)
	}
	if !isFlagPassed("download-timeout") && config["download-timeout"] != nil {
		downloadTimeout = config["download-timeout"].(int)
	}

	if !isFlagPassed("retry") && config["retry"] != nil {
		retry = config["retry"].(int)
	}
	if !isFlagPassed("retry-interval") && config["retry-interval"] != nil {
		retryInterval = config["retry-interval"].(float64)
	}
	if !isFlagPassed("max-download-parallel") && config["max-download-parallel"] != nil {
		maxDownloadParallel = config["max-download-parallel"].(int)
	}
	if !isFlagPassed("rate-limit") && config["rate-limit"] != nil {
		rateLimit = config["rate-limit"].(int)
	}
}
