# Kemono-scraper

A simple downloader to download images from kemono.cr

## Flag option

### Cookie file

only needed if you want to download favorite creators or posts

`--cookie PATH` cookie file, default is cookies.txt (value separate by whitespace) syntax:

| Domain        | Include subdomains | Path | Secure | Expiry     | Name        | Value   |
|---------------|--------------------|------|--------|------------|-------------|---------|
| .kemono.cr    | FALSE              | /    | TRUE   | 1706755572 | kemono_auth | <value> |

you can get cookies easily by using Chrome extension [Get cookies.txt LOCALLY](https://chrome.google.com/webstore/detail/get-cookiestxt-locally/cclelndahbckbenkjhflpdbgdldlbecc)

### Windows

Windows can detect the cookie file automatically (Not supported in no_cookies_detection version)

`--cookie-browser string` which browser to use, default is chrome (supported: chrome, firefox, edge , opera, vivaldi)

### Download Options

`--link [<urls>]`: download link, separate by comma

`--creator [<service>:<id>]`: download creators, separate by comma

`--banner bool`: download banner, default is false (kemono only)

`--fav-site string`: specify the website to get favorites from (kemono or coomer), separated by comma

`--fav-creator bool`: download favorite creator, default is false

`--fav-post bool` download favorite post, default is false

### Post Filter Options

`--first int`: download first n post

`--last int`: download last n post

`--date YYYYMMDD`: download post on date

`--date-before YYYYMMDD`: download post before date

`--date-after YYYYMMDD`: download post after date

`--update YYYYMMDD`: download post updated on date

`--update-before YYYYMMDD`: download post updated before date

`--update-after YYYYMMDD`: download post updated after date

### Image Filter Options

`--extension-only [<ext>]`: download post with extension, separate by comma

`--extension-exclude [<ext>]`: download post without extension, separate by comma

`--max-size string`: download post with size less than max-size (e.g. 1 MB, 1KB, 1.5 gb, etc.)

`--min-size string`: download post with size greater than min-size (e.g. 1 MB, 1KB, 1.5 gb, etc.)

## Downloader options

`--output PATH`: output path

`--template <tags>`: The template for customizing download paths, where you can use the following keywords to specify different parts of the path:

- `<ks:service>`: creator service
- `<ks:creator>`: creator name
- `<ks:post>`: post title
- `<ks:index>`: file index
- `<ks:filename>`: file name
- `<ks:filehash>`: file hash
- `<ks:extension>`: file extension

For example:

`[<ks:service>] <ks:creator>/<ks:post>/<ks:index>-<ks:filename><ks:extension>`

`--image-template <tags>` The template for customizing image file, `--template` should be set first.

`--video-template <tags>` The template for customizing video file, `--template` should be set first.

`--audio-template <tags>` The template for customizing audio file, `--template` should be set first.

`--archive-template <tags>` The template for customizing archive file, `--template` should be set first.

`--content bool`: download content, default is false

`--overwrite bool`: overwrite existing file

`--async bool`: download posts asynchronously, may cause the file order is not the same as the post order, can be used with --with-prefix-number, default false

`--max-download-parallel int`: max download file concurrent, default is 3, async mode only

`--with-prefix-number bool`: add prefix number to file name `<order>_<filename>`, default false

`--name-rule-only-index bool`: only use index as file name, default false

`--download-timeout int`: download timeout in seconds, default 1800

`--retry int`: retry times, default 3

`--retry-interval number`: retry interval in seconds, default 10. The number can be specified as either an int or float type

`--rate-limit int`: rate limit in request/s, default 2

`--proxy string`: proxy url, default is empty, support socks5, http, https (e.g. socks5://proxy:1080)

## Config File

config file is in `./config.yaml`

Options in config file are the same as command-line flag options, but will be overridden by flags (if both exists).
Usually used for setting the default settings for the scraper.

```yaml
banner: true
async: true
max-download-parallel: 5
output: ./downloads
template: "[<ks:service>] <ks:creator>/<ks:post>/<ks:filename><ks:extension>"
image-template: "[<ks:service>] <ks:creator>/<ks:post>/<ks:index><ks:extension>"
video-template: "[<ks:service>] <ks:creator>/<ks:post>/video/<ks:filename><ks:extension>"
retry: 10
retry-interval: 15
# proxy: socks5://proxy:1080
```

## Build from Source

Cloning the repository:

```bash
git clone https://github.com/elvis972602/Kemono-scraper
cd Kemono-scraper/main
```

Download all the dependencies:

```bash
go mod tidy
```

Build the project:

```bash
go build
```

- No cookies detection:

```bash
go build -tags=no_cookies_detection
```

## Features

With Kemono-scraper, you can implement a Downloader to take advantage of features such as multi-connection downloading, resume broken downloads, and more.
