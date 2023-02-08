package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/elvis972602/kemono-scraper/downloader"
	"github.com/elvis972602/kemono-scraper/kemono"
	"github.com/elvis972602/kemono-scraper/term"
	"github.com/elvis972602/kemono-scraper/utils"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	Kemono = "kemono"
	Coomer = "coomer"
)

var (
	creators               []string
	links                  []string
	options                map[string][]kemono.Option
	sharedOptions          []kemono.Option
	downloaderOptions      []downloader.DownloadOption
	hasLink                bool
	s, srv, userId, postId string
	// map[<Creator>][]<postId>
	idFilter map[string]map[kemono.Creator][]string
)

func init() {
	idFilter = make(map[string]map[kemono.Creator][]string)
	idFilter[Kemono] = make(map[kemono.Creator][]string)
	idFilter[Coomer] = make(map[kemono.Creator][]string)
	options = make(map[string][]kemono.Option)
}

func main() {
	flag.Parse()

	if help {
		PrintDefaults()
		return
	}

	setFlag()

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
		users := make(map[string][]string)
		for _, l := range links {
			s, srv, userId, postId = parasLink(l)

			cs := kemono.NewCreator(srv, userId)
			users[s] = append(users[s], srv, userId)
			if postId != "" {
				idFilter[s][cs] = append(idFilter[s][cs], postId)
			}
			options[s] = append(options[s],
				kemono.WithUsersPair(users[s]...),
			)
		}
	}

	// check creator
	if len(creators) > 0 {
		for _, c := range creators {
			creatorComponents := strings.Split(c, ":")
			if len(creatorComponents) != 2 {
				log.Fatalf("invalid creator %s", c)
			}
			s, ok := kemono.SiteMap[creatorComponents[0]]
			if !ok {
				log.Fatalf("invalid creator %s", c)
			}
			options[s] = append(options[s], kemono.WithUsersPair(creatorComponents[0], creatorComponents[1]))

		}
	} else if !hasLink && !favoriteCreator && !favoritePost {
		log.Fatal("creator is empty")
	}

	if favoriteCreator || favoritePost {
		if site == "" {
			log.Fatal("fav-site is empty")
		}
		siteComponents := strings.Split(site, ",")
		for _, siteComponent := range siteComponents {
			siteComponent = strings.TrimSpace(siteComponent)
			if siteComponent == "" {
				continue
			}
			cs := getCookies(siteComponent)
			if len(cs) == 0 {
				log.Fatal("cookie is empty")
			}
			if favoriteCreator {
				for _, c := range fetchFavoriteCreators(siteComponent, cs) {
					options[siteComponent] = append(options[siteComponent], kemono.WithUsersPair(c.Service, c.Id))
				}
			}
			if favoritePost {
				for _, c := range fetchFavoritePosts(siteComponent, cs) {
					u := kemono.NewCreator(c.Service, c.User)
					options[siteComponent] = append(options[siteComponent], kemono.WithUsers(u))
					idFilter[siteComponent][u] = append(idFilter[siteComponent][u], c.Id)
				}
			}
		}
	}

	for k, v := range idFilter {
		for u, ids := range v {
			if len(ids) > 0 {
				options[k] = append(options[k], kemono.WithUserPostFilter(u,
					kemono.IdFilter(ids...),
				))
			}
		}
	}

	// banner
	sharedOptions = append(sharedOptions, kemono.WithBanner(banner))

	// overwrite
	downloaderOptions = append(downloaderOptions, downloader.OverWrite(overwrite))

	// check first
	if first != 0 {
		sharedOptions = append(sharedOptions, kemono.WithPostFilter(
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
		sharedOptions = append(sharedOptions, kemono.WithPostFilter(
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
		sharedOptions = append(sharedOptions, kemono.WithPostFilter(
			kemono.ReleaseDateFilter(t.Add(-1), t.Add(24*time.Hour-1)),
		))
	}

	if dateBefore != 0 {
		t := parasData(strconv.Itoa(dateBefore))
		sharedOptions = append(sharedOptions, kemono.WithPostFilter(
			kemono.ReleaseDateBeforeFilter(t),
		))
	}

	if dateAfter != 0 {
		t := parasData(strconv.Itoa(dateAfter))
		sharedOptions = append(sharedOptions, kemono.WithPostFilter(
			kemono.ReleaseDateAfterFilter(t),
		))
	}

	if update != 0 {
		t := parasData(strconv.Itoa(update))
		sharedOptions = append(sharedOptions, kemono.WithPostFilter(
			kemono.EditDateFilter(t.Add(-1), t.Add(24*time.Hour-1)),
		))
	}

	if updateBefore != 0 {
		t := parasData(strconv.Itoa(updateBefore))
		sharedOptions = append(sharedOptions, kemono.WithPostFilter(
			kemono.EditDateBeforeFilter(t),
		))
	}

	if updateAfter != 0 {
		t := parasData(strconv.Itoa(updateAfter))
		sharedOptions = append(sharedOptions, kemono.WithPostFilter(
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
		sharedOptions = append(sharedOptions, kemono.WithAttachmentFilter(
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
		sharedOptions = append(sharedOptions, kemono.WithAttachmentFilter(
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

	if maxSize != "" {
		size := utils.ParseSize(maxSize)
		downloaderOptions = append(downloaderOptions, downloader.MaxSize(size))
	}

	if minSize != "" {
		size := utils.ParseSize(minSize)
		downloaderOptions = append(downloaderOptions, downloader.MinSize(size))
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
	sharedOptions = append(sharedOptions, kemono.SetLog(terminal))

	var (
		KKemono          *kemono.Kemono
		KCoomer          *kemono.Kemono
		KemonoDownloader kemono.Downloader
		CoomerDownloader kemono.Downloader
		k, c             bool
	)

	// Kemono
	if len(options[Kemono]) > 0 {
		k = true
		options[Kemono] = append(options[Kemono], sharedOptions...)
		options[Kemono] = append(options[Kemono], kemono.WithDomain("kemono"))
		downloaderOptions = append(downloaderOptions, downloader.BaseURL("https://kemono.party"))
		KemonoDownloader = downloader.NewDownloader(downloaderOptions...)
		options[Kemono] = append(options[Kemono], kemono.SetDownloader(KemonoDownloader))
		KKemono = kemono.NewKemono(options[Kemono]...)
	}
	if len(options[Coomer]) > 0 {
		c = true
		options[Coomer] = append(options[Coomer], sharedOptions...)
		options[Coomer] = append(options[Coomer], kemono.WithDomain("coomer"))
		downloaderOptions = append(downloaderOptions, downloader.BaseURL("https://coomer.party"))
		CoomerDownloader = downloader.NewDownloader(downloaderOptions...)
		options[Coomer] = append(options[Coomer], kemono.SetDownloader(CoomerDownloader))
		options[Coomer] = append(options[Coomer], kemono.WithBanner(true))
		KCoomer = kemono.NewKemono(options[Coomer]...)
	}

	if k {
		terminal.Print("Downloading Kemono")
		KKemono.Start()
	}
	if c {
		terminal.Print("Downloading Coomer")
		KCoomer.Start()
	}

	defer func() {
		ctx.Done()
	}()

}

func parasLink(link string) (s, service, userId, postId string) {
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

	s = hostComponents[0]
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
	if !isFlagPassed("fav-site") && config["fav-site"] != nil {
		site = config["fav-site"].(string)
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
	if !isFlagPassed("max-size") && config["max-size"] != nil {
		maxSize = config["max-size"].(string)
	}
	if !isFlagPassed("min-size") && config["min-size"] != nil {
		minSize = config["min-size"].(string)
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
	if !isFlagPassed("fav-creator") && config["fav-creator"] != nil {
		favoriteCreator = config["fav-creator"].(bool)
	}
	if !isFlagPassed("fav-post") && config["fav-post"] != nil {
		favoritePost = config["fav-post"].(bool)
	}
	if !isFlagPassed("cookie-browser") && config["cookie-browser"] != nil {
		cookieBrowser = config["cookie-browser"].(string)
	}
	if !isFlagPassed("cookie") && config["cookie"] != nil {
		cookieFile = config["cookie"].(string)
	}

}

func DirectoryName(p kemono.Post) string {
	return fmt.Sprintf("[%s] [%s] %s", p.Published.Format("20060102"), p.Id, p.Title)
}

func fetchFavoriteCreators(s string, cookie []*http.Cookie) []kemono.FavoriteCreator {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s.party/api/favorites?type=user", s), nil)
	if err != nil {
		log.Fatalf("Error creating request: %s", err)
	}
	for _, v := range cookie {
		req.AddCookie(v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Error getting favorites: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Fatalf("Error getting favorites: %d", resp.StatusCode)
	}
	var creators []kemono.FavoriteCreator
	err = json.NewDecoder(resp.Body).Decode(&creators)
	if err != nil {
		log.Fatalf("Error decoding favorites: %s", err)
	}
	return creators
}

func fetchFavoritePosts(s string, cookie []*http.Cookie) []kemono.PostRaw {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s.party/api/favorites?type=post", s), nil)
	if err != nil {
		log.Fatalf("Error creating request: %s", err)
	}
	for _, v := range cookie {
		req.AddCookie(v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Error getting posts: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Fatalf("Error getting posts: %d", resp.StatusCode)
	}
	var posts []kemono.PostRaw
	err = json.NewDecoder(resp.Body).Decode(&posts)
	if err != nil {
		log.Fatalf("Error decoding posts: %s", err)
	}
	return posts
}

func parasCookeiFile(cookieFile string) []*http.Cookie {
	var (
		cookies []*http.Cookie
		domain  string
	)
	f, err := os.Open(cookieFile)
	if err != nil {
		log.Fatalf("Error opening cookie file: %s", err)
	}
	defer f.Close()
	if site != "" {
		domain = site
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		columns := strings.Fields(line)
		if len(columns) < 7 {
			continue
		}
		domainComponents := strings.Split(columns[0], ".")
		d := domainComponents[len(domainComponents)-2]
		if domain == "" {
			domain = d
		} else if domain != d {
			// other domain ignore
			continue
		}
		exp, err := strconv.ParseInt(columns[4], 10, 64)
		if err != nil {
			continue
		}
		c := &http.Cookie{
			Name:    columns[5],
			Value:   columns[6],
			Domain:  columns[0],
			Path:    columns[2],
			Secure:  columns[3] == "TRUE",
			Expires: time.Unix(exp, 0),
		}
		cookies = append(cookies, c)
	}
	if site == "" {
		site = domain
	}
	return cookies
}
