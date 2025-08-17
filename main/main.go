package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	tmpl "text/template"
	"time"

	"github.com/elvis972602/kemono-scraper/downloader"
	"github.com/elvis972602/kemono-scraper/kemono"
	"github.com/elvis972602/kemono-scraper/term"
	"github.com/elvis972602/kemono-scraper/utils"
	"github.com/mattn/go-colorable"
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
	setPassedFlags()

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

	if output == "" {
		output = "./download"
	}

	if template == "" {
		if imageTemplate != "" || videoTemplate != "" || audioTemplate != "" || archiveTemplate != "" {
			log.Printf("to use image/video/audio/archive template, you must set template first")
		}
		var t *tmpl.Template
		defaultTemp, err := LoadPathTmpl(TmplDefault, output)
		if err != nil {
			log.Fatalf("load template failed: %s", err)
		}
		if nameRuleOnlyIndex {
			t, err = LoadPathTmpl(TmplIndexNumber, output)
			if err != nil {
				log.Fatalf("load template failed: %s", err)
			}
		} else if withPrefixNumber {
			t, err = LoadPathTmpl(TmplWithPrefixNumber, output)
			if err != nil {
				log.Fatalf("load template failed: %s", err)
			}
		} else {
			t = defaultTemp
		}
		downloaderOptions = append(downloaderOptions, downloader.SavePath(func(creator kemono.Creator, post kemono.Post, i int, attachment kemono.File) string {
			ext := filepath.Ext(attachment.Name)
			filehash := filepath.Base(attachment.Path)[0 : len(filepath.Base(attachment.Path))-len(filepath.Ext(attachment.Path))]
			filename := attachment.Name[0 : len(attachment.Name)-len(ext)]
			// use Path extension if Name extension is empty
			if ext == "" {
				ext = filepath.Ext(attachment.Path)
			}
			pathConfig := &PathConfig{
				Service:   creator.Service,
				Creator:   utils.ValidDirectoryName(creator.Name),
				Post:      utils.ValidDirectoryName(DirectoryName(post)),
				Index:     i,
				Filename:  utils.ValidDirectoryName(filename),
				Filehash:  utils.ValidDirectoryName(filehash),
				Extension: ext,
			}
			if ext == ".zip" || ext == ".rar" || ext == ".7z" {
				return ExecutePathTmpl(defaultTemp, pathConfig)
			} else {
				return ExecutePathTmpl(t, pathConfig)
			}
		}))
	} else {
		tmplCache := NewTmplCache()
		tmplCache.init()

		downloaderOptions = append(downloaderOptions, downloader.SavePath(func(creator kemono.Creator, post kemono.Post, i int, attachment kemono.File) string {
			ext := filepath.Ext(attachment.Name)
			filehash := filepath.Base(attachment.Path)[0 : len(filepath.Base(attachment.Path))-len(filepath.Ext(attachment.Path))]
			filename := attachment.Name[0 : len(attachment.Name)-len(ext)]
			// use Path extension if Name extension is empty
			if ext == "" {
				ext = filepath.Ext(attachment.Path)
			}
			pathConfig := &PathConfig{
				Service:   creator.Service,
				Creator:   utils.ValidDirectoryName(creator.Name),
				Post:      utils.ValidDirectoryName(DirectoryName(post)),
				Index:     i,
				Filename:  utils.ValidDirectoryName(filename),
				Filehash:  utils.ValidDirectoryName(filehash),
				Extension: ext,
			}
			return tmplCache.Execute(getTyp(ext), pathConfig)
		}))
	}

	downloaderOptions = append(downloaderOptions, downloader.WithContent(content))

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
		sharedOptions = append(sharedOptions, kemono.SetRetry(retry))
	}

	if retryInterval < 0 {
		log.Fatalf("retry interval must be greater than 0")
	} else {
		downloaderOptions = append(downloaderOptions, downloader.RetryInterval(time.Duration(retryInterval)*time.Second))
		sharedOptions = append(sharedOptions, kemono.SetRetryInterval(time.Duration(retryInterval)*time.Second))
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

	if proxy != "" {
		downloaderOptions = append(downloaderOptions, downloader.WithProxy(proxy))
	}

	ctx := context.Background()
	defer ctx.Done()

	terminal := term.NewTerminal(colorable.NewColorableStdout(), colorable.NewColorableStderr(), false)
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

		kemonoHost := siteHost(Kemono)
		downloaderOptions = append(downloaderOptions, downloader.BaseURL("https://"+kemonoHost))
		token, err := utils.GenerateToken(16)
		if err != nil {
			log.Fatalf("generate token failed: %s", err)
		}
		downloaderOptions = append(downloaderOptions, downloader.WithCookie([]*http.Cookie{
			{
				Name:   "__ddg2",
				Value:  token,
				Path:   "/",
				Domain: "." + kemonoHost,
				Secure: false,
			},
		}))
		downloaderOptions = append(downloaderOptions, downloader.WithHeader(downloader.Header{
			"Host":                      kemonoHost,
			"User-Agent":                downloader.UserAgent,
			"Referer":                   "https://" + kemonoHost,
			"Accept":                    downloader.Accept,
			"Accept-Language":           downloader.AcceptLanguage,
			"Accept-Encoding":           downloader.AcceptEncoding,
			"Sec-Ch-Ua":                 downloader.SecChUA,
			"Sec-Ch-Ua-Mobile":          downloader.SecChUAMobile,
			"Sec-Fetch-Dest":            downloader.SecFetchDest,
			"Sec-Fetch-Mode":            downloader.SecFetchMode,
			"Sec-Fetch-Site":            downloader.SecFetchSite,
			"Sec-Fetch-User":            downloader.SecFetchUser,
			"Upgrade-Insecure-Requests": downloader.UpgradeInsecureRequests,
			"Connection":                "keep-alive",
		}))
		KemonoDownloader = downloader.NewDownloader(downloaderOptions...)
		options[Kemono] = append(options[Kemono], kemono.SetDownloader(KemonoDownloader))
		KKemono = kemono.NewKemono(options[Kemono]...)
	}
	if len(options[Coomer]) > 0 {
		c = true
		options[Coomer] = append(options[Coomer], sharedOptions...)
		options[Coomer] = append(options[Coomer], kemono.WithDomain("coomer"))

		coomerHost := siteHost(Coomer)
		downloaderOptions = append(downloaderOptions, downloader.BaseURL("https://"+coomerHost))
		token, err := utils.GenerateToken(16)
		if err != nil {
			log.Fatalf("generate token failed: %s", err)
		}
		downloaderOptions = append(downloaderOptions, downloader.WithCookie([]*http.Cookie{
			{
				Name:   "__ddg2",
				Value:  token,
				Path:   "/",
				Domain: "." + coomerHost,
			},
		}))
		downloaderOptions = append(downloaderOptions, downloader.WithHeader(downloader.Header{
			"Host":                      coomerHost,
			"User-Agent":                downloader.UserAgent,
			"Referer":                   "https://" + coomerHost + "/",
			"Accept":                    downloader.Accept,
			"Accept-Language":           downloader.AcceptLanguage,
			"Accept-Encoding":           downloader.AcceptEncoding,
			"Sec-Ch-Ua":                 downloader.SecChUA,
			"Sec-Ch-Ua-Mobile":          downloader.SecChUAMobile,
			"Sec-Fetch-Dest":            downloader.SecFetchDest,
			"Sec-Fetch-Mode":            downloader.SecFetchMode,
			"Sec-Fetch-Site":            downloader.SecFetchSite,
			"Sec-Fetch-User":            downloader.SecFetchUser,
			"Upgrade-Insecure-Requests": downloader.UpgradeInsecureRequests,
			"Connection":                "keep-alive",
		}))
		CoomerDownloader = downloader.NewDownloader(downloaderOptions...)
		options[Coomer] = append(options[Coomer], kemono.SetDownloader(CoomerDownloader))
		options[Coomer] = append(options[Coomer], kemono.WithBanner(true))
		KCoomer = kemono.NewKemono(options[Coomer]...)
	}

	if k {
		terminal.Print("Downloading Kemono")
		err := KKemono.Start()
		if err != nil {
			log.Printf("kemono start failed: %s", err)
		}
	}
	if c {
		terminal.Print("Downloading Coomer")
		err := KCoomer.Start()
		if err != nil {
			log.Printf("coomer start failed: %s", err)
		}
	}
}

func parasLink(link string) (s, service, userId, postId string) {
	u, err := url.Parse(link)
	if err != nil {
		log.Fatal("invalid url")
	}

	// Accept kemono/coomer with su/cr/st TLDs (and optional subdomains)
	pattern := `(?i)^(?:.*\.)?(kemono|coomer)\.(su|cr|st)$`
	re := regexp.MustCompile(pattern)

	matchedSubstrings := re.FindStringSubmatch(u.Host)

	if matchedSubstrings == nil {
		log.Fatal("invalid host:", u.Host)
	}

	s = matchedSubstrings[1]

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
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func setFlag() {
	if !passedFlags["link"] && config["link"] != nil {
		link = config["link"].(string)
	}
	if !passedFlags["fav-site"] && config["fav-site"] != nil {
		site = config["fav-site"].(string)
	}
	if !passedFlags["creator"] && config["creator"] != nil {
		creator = config["creator"].(string)
	}
	if !passedFlags["banner"] && config["banner"] != nil {
		banner = config["banner"].(bool)
	}
	if !passedFlags["overwrite"] && config["overwrite"] != nil {
		overwrite = config["overwrite"].(bool)
	}
	if !passedFlags["first"] && config["first"] != nil {
		first = config["first"].(int)
	}
	if !passedFlags["last"] && config["last"] != nil {
		last = config["last"].(int)
	}
	if !passedFlags["date"] && config["date"] != nil {
		date = config["date"].(int)
	}
	if !passedFlags["date-before"] && config["date-before"] != nil {
		dateBefore = config["date-before"].(int)
	}
	if !passedFlags["date-after"] && config["date-after"] != nil {
		dateAfter = config["date-after"].(int)
	}
	if !passedFlags["update"] && config["update"] != nil {
		update = config["update"].(int)
	}
	if !passedFlags["update-before"] && config["update-before"] != nil {
		updateBefore = config["update-before"].(int)
	}
	if !passedFlags["update-after"] && config["update-after"] != nil {
		updateAfter = config["update-after"].(int)
	}
	if !passedFlags["extension-only"] && config["extension-only"] != nil {
		extensionOnly = config["extension-only"].(string)
	}
	if !passedFlags["extension-exclude"] && config["extension-exclude"] != nil {
		extensionExclude = config["extension-exclude"].(string)
	}
	if !passedFlags["output"] && config["output"] != nil {
		output = config["output"].(string)
	}
	if !passedFlags["template"] && config["template"] != nil {
		template = config["template"].(string)
	}
	if !passedFlags["image-template"] && config["image-template"] != nil {
		imageTemplate = config["image-template"].(string)
	}
	if !passedFlags["audio-template"] && config["audio-template"] != nil {
		audioTemplate = config["audio-template"].(string)
	}
	if !passedFlags["video-template"] && config["video-template"] != nil {
		videoTemplate = config["video-template"].(string)
	}
	if !passedFlags["archive-template"] && config["archive-template"] != nil {
		archiveTemplate = config["archive-template"].(string)
	}
	if !passedFlags["contrnt"] && config["content"] != nil {
		content = config["content"].(bool)
	}
	if !passedFlags["async"] && config["async"] != nil {
		async = config["async"].(bool)
	}
	if !passedFlags["max-size"] && config["max-size"] != nil {
		maxSize = config["max-size"].(string)
	}
	if !passedFlags["min-size"] && config["min-size"] != nil {
		minSize = config["min-size"].(string)
	}
	if !passedFlags["with-prefix-number"] && config["with-prefix-number"] != nil {
		withPrefixNumber = config["with-prefix-number"].(bool)
	}
	if !passedFlags["name-rule-only-index"] && config["name-rule-only-index"] != nil {
		nameRuleOnlyIndex = config["name-rule-only-index"].(bool)
	}
	if !passedFlags["download-timeout"] && config["download-timeout"] != nil {
		downloadTimeout = config["download-timeout"].(int)
	}

	if !passedFlags["retry"] && config["retry"] != nil {
		retry = config["retry"].(int)
	}
	if !passedFlags["retry-interval"] && config["retry-interval"] != nil {
		// check if retry-interval is float64 or int
		_, ok := config["retry-interval"].(float64)
		if !ok {
			retryInterval = float64(config["retry-interval"].(int))
		} else {
			retryInterval = config["retry-interval"].(float64)
		}
	}
	if !passedFlags["max-download-parallel"] && config["max-download-parallel"] != nil {
		maxDownloadParallel = config["max-download-parallel"].(int)
	}
	if !passedFlags["rate-limit"] && config["rate-limit"] != nil {
		rateLimit = config["rate-limit"].(int)
	}
	if !passedFlags["proxy"] && config["proxy"] != nil {
		proxy = config["proxy"].(string)
	}
	if !passedFlags["fav-creator"] && config["fav-creator"] != nil {
		favoriteCreator = config["fav-creator"].(bool)
	}
	if !passedFlags["fav-post"] && config["fav-post"] != nil {
		favoritePost = config["fav-post"].(bool)
	}
	if !passedFlags["cookie-browser"] && config["cookie-browser"] != nil {
		cookieBrowser = config["cookie-browser"].(string)
	}
	if !passedFlags["cookie"] && config["cookie"] != nil {
		cookieFile = config["cookie"].(string)
	}
}

func DirectoryName(p kemono.Post) string {
	return fmt.Sprintf("[%s] [%s] %s", p.Published.Format("20060102"), p.Id, p.Title)
}

func fetchFavoriteCreators(s string, cookie []*http.Cookie) []kemono.FavoriteCreator {
	host := siteHost(s)
	log.Printf("fetching favorite creators from %s", host)
	var client *http.Client
	client = http.DefaultClient
	if proxy != "" {
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
				ResponseHeaderTimeout: 30 * time.Second,
			},
		}
		downloader.AddProxy(proxy, client.Transport.(*http.Transport))
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s/api/v1/account/favorites?type=user", host), nil)
	if err != nil {
		log.Fatalf("Error creating request: %s", err)
	}
	req.Header.Set("Host", host)
	for _, v := range cookie {
		req.AddCookie(v)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error getting favorites: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Fatalf("Error getting favorites: %d", resp.StatusCode)
	}
	var favoriteCreators []kemono.FavoriteCreator
	err = json.NewDecoder(resp.Body).Decode(&favoriteCreators)
	if err != nil {
		log.Fatalf("Error decoding favorites: %s", err)
	}
	return favoriteCreators
}

func fetchFavoritePosts(s string, cookie []*http.Cookie) []kemono.PostRaw {
	host := siteHost(s)
	log.Printf("fetching favorite posts from %s", host)
	var client *http.Client
	client = http.DefaultClient
	if proxy != "" {
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
				ResponseHeaderTimeout: 30 * time.Second,
			},
		}
		downloader.AddProxy(proxy, client.Transport.(*http.Transport))
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s/api/v1/account/favorites?type=post", host), nil)
	if err != nil {
		log.Fatalf("Error creating request: %s", err)
	}
	req.Header.Set("Host", host)
	for _, v := range cookie {
		req.AddCookie(v)
	}
	resp, err := client.Do(req)
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

func parasCookieFile(cookieFile string) []*http.Cookie {
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

// siteHost resolves the canonical host for a given site identifier ("kemono" or "coomer").
// Updated to support kemono.cr and coomer.st.
func siteHost(s string) string {
	switch strings.ToLower(s) {
	case Kemono:
		return "kemono.cr"
	case Coomer:
		return "coomer.st"
	default:
		// Fallback: if a full host was passed in, return as-is.
		return s
	}
}
