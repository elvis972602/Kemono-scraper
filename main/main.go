package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/elvis972602/kemono-scraper/downloader"
	"github.com/elvis972602/kemono-scraper/kemono"
	"github.com/elvis972602/kemono-scraper/term"
	"github.com/elvis972602/kemono-scraper/utils"
	"log"
	"net/url"
	"os"
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
		downloaderOptions      []downloader.DownloadOption
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
	downloaderOptions = append(downloaderOptions, downloader.Async(async))

	if len(links) > 0 {
		hasLink = true
		users := make([]string, 0)
		ids := make(map[string][]string, 0)
		for i, l := range links {
			s, srv, userId, postId = parasLink(l)
			if i == 0 {
				site = s
			} else {
				if s != site {
					log.Fatalf("site %s is not match %s", s, site)
				}
			}
			cs := kemono.NewCreator(srv, userId).PairString()
			users = append(users, srv, userId)
			if postId != "" {
				ids[cs] = append(ids[cs], postId)
			}
		}
		options = append(options,
			kemono.WithDomain(s),
			kemono.WithUsersPair(users...),
		)
		for i := 0; i < len(users); i += 2 {
			cs := kemono.NewCreator(users[i], users[i+1]).PairString()
			if len(ids[cs]) == 0 {
				continue
			}
			options = append(options,
				kemono.WithUserPostFilter(kemono.NewCreator(users[i], users[i+1]),
					kemono.IdFilter(ids[cs]...)))
		}
		downloaderOptions = append(downloaderOptions, downloader.BaseURL(fmt.Sprintf("https://%s.party", s)))
	}

	// check creator
	if len(creators) > 0 {
		users := make([]string, 0)
		for i, c := range creators {
			creatorComponents := strings.Split(c, ":")
			if len(creatorComponents) != 2 {
				log.Fatalf("invalid creator %s", c)
			}
			if i == 0 && site == "" {
				site = kemono.SiteMap[creatorComponents[0]]
			} else {
				if site != kemono.SiteMap[creatorComponents[0]] {
					log.Fatalf("site %s not match creator %s", site, creatorComponents[0])
				}
			}
			users = append(users, creatorComponents[0], creatorComponents[1])
		}
		options = append(options, kemono.WithUsersPair(users...))
		if !hasLink {
			options = append(options, kemono.WithDomain(site))
			downloaderOptions = append(downloaderOptions, downloader.BaseURL(fmt.Sprintf("https://%s.party", site)))
		}
	} else if !hasLink {
		log.Fatal("creator is empty")
	}

	// banner
	options = append(options, kemono.WithBanner(banner))

	// overwrite
	downloaderOptions = append(downloaderOptions, downloader.OverWrite(overwrite))

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

	if date != 0 {
		t := parasData(strconv.Itoa(date))
		options = append(options, kemono.WithPostFilter(
			kemono.ReleaseDateFilter(t.Add(-1), t.Add(24*time.Hour-1)),
		))
	}

	if dateBefore != 0 {
		t := parasData(strconv.Itoa(dateBefore))
		options = append(options, kemono.WithPostFilter(
			kemono.ReleaseDateBeforeFilter(t),
		))
	}

	if dateAfter != 0 {
		t := parasData(strconv.Itoa(dateAfter))
		options = append(options, kemono.WithPostFilter(
			kemono.ReleaseDateAfterFilter(t),
		))
	}

	if update != 0 {
		t := parasData(strconv.Itoa(update))
		options = append(options, kemono.WithPostFilter(
			kemono.EditDateFilter(t.Add(-1), t.Add(24*time.Hour-1)),
		))
	}

	if updateBefore != 0 {
		t := parasData(strconv.Itoa(updateBefore))
		options = append(options, kemono.WithPostFilter(
			kemono.EditDateBeforeFilter(t),
		))
	}

	if updateAfter != 0 {
		t := parasData(strconv.Itoa(updateAfter))
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
				return fmt.Sprintf(filepath.Join("%s", "%s", "%s", "%s"), output, utils.ValidDirectoryName(creator.Name), utils.ValidDirectoryName(DirectoryName(post)), utils.ValidDirectoryName(name))
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
				return fmt.Sprintf(filepath.Join("%s", "%s", "%s", "%s"), output, utils.ValidDirectoryName(creator.Name), utils.ValidDirectoryName(DirectoryName(post)), utils.ValidDirectoryName(name))
			}
		}
		if !ruleflag {
			pathFunc = func(creator kemono.Creator, post kemono.Post, i int, attachment kemono.File) string {
				return fmt.Sprintf(filepath.Join("%s", "%s", "%s", "%s"), output, utils.ValidDirectoryName(creator.Name), utils.ValidDirectoryName(DirectoryName(post)), utils.ValidDirectoryName(attachment.Name))
			}
		}
		downloaderOptions = append(downloaderOptions, downloader.SavePath(pathFunc))
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
				return fmt.Sprintf(filepath.Join("./download", "%s", "%s", "%s"), utils.ValidDirectoryName(creator.Name), utils.ValidDirectoryName(DirectoryName(post)), utils.ValidDirectoryName(name))
			}
			downloaderOptions = append(downloaderOptions, downloader.SavePath(pathFunc))
		}

		if nameRuleOnlyIndex {
			var pathFunc func(creator kemono.Creator, post kemono.Post, i int, attachment kemono.File) string
			pathFunc = func(creator kemono.Creator, post kemono.Post, i int, attachment kemono.File) string {
				ext := filepath.Ext(attachment.Name)
				name := fmt.Sprintf("%d%s", i, ext)
				return fmt.Sprintf(filepath.Join("./download", "%s", "%s", "%s"), utils.ValidDirectoryName(creator.Name), utils.ValidDirectoryName(DirectoryName(post)), utils.ValidDirectoryName(name))
			}
			downloaderOptions = append(downloaderOptions, downloader.SavePath(pathFunc))
		}
	}

	if downloadTimeout <= 0 {
		log.Fatalf("invalid download timeout %d", downloadTimeout)
	} else {
		downloaderOptions = append(downloaderOptions, downloader.Timeout(time.Duration(downloadTimeout)*time.Second))
	}

	if retry < 0 {
		log.Fatalf("retry must be greater than 0")
	} else {
		downloaderOptions = append(downloaderOptions, downloader.Retry(retry))
	}

	if retryInterval < 0 {
		log.Fatalf("retry interval must be greater than 0")
	} else {
		downloaderOptions = append(downloaderOptions, downloader.RetryInterval(time.Duration(retryInterval)*time.Second))
	}

	// check maxDownloadGoroutine
	if maxDownloadParallel <= 0 {
		log.Fatalf("maxDownloadParallel must be greater than 0")
	} else {
		downloaderOptions = append(downloaderOptions, downloader.MaxConcurrent(maxDownloadParallel))
	}

	if rateLimit <= 0 {
		log.Fatalf("rate limit must be greater than 0")
	} else {
		downloaderOptions = append(downloaderOptions, downloader.RateLimit(rateLimit))
	}

	ctx := context.Background()

	terminal := term.NewTerminal(os.Stdout, os.Stderr, false)
	go terminal.Run(ctx)

	downloaderOptions = append(downloaderOptions, downloader.SetLog(terminal))
	options = append(options, kemono.SetLog(terminal))

	download := downloader.NewDownloader(downloaderOptions...)

	options = append(options, kemono.SetDownloader(download))

	K := kemono.NewKemono(options...)

	K.Start()

	defer func() {
		ctx.Done()
	}()

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
		log.Fatalf("invalid date %s", data)
	}
	year, err := strconv.Atoi(data[:4])
	if err != nil {
		log.Fatalf("invalid date %s", data)
	}
	month, err := strconv.Atoi(data[4:6])
	if err != nil {
		log.Fatalf("invalid date %s", data)
	}
	day, err := strconv.Atoi(data[6:])
	if err != nil {
		log.Fatalf("invalid date %s", data)
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
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
	if !isFlagPassed("date") && config["date"] != nil {
		date = config["date"].(int)
	}
	if !isFlagPassed("date-before") && config["date-before"] != nil {
		date = config["date-before"].(int)
	}
	if !isFlagPassed("date-after") && config["date-after"] != nil {
		date = config["date-after"].(int)
	}
	if !isFlagPassed("update") && config["update"] != nil {
		update = config["update"].(int)
	}
	if !isFlagPassed("update-before") && config["update-before"] != nil {
		update = config["update-before"].(int)
	}
	if !isFlagPassed("update-after") && config["update-after"] != nil {
		update = config["update-after"].(int)
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

func DirectoryName(p kemono.Post) string {
	return fmt.Sprintf("[%s][%s]%s", p.Published.Format("20060102"), p.Id, p.Title)
}
