# Kemono-scraper
A simple downloader to  download images from kemono.party

## Flag Option

### Cookie File
only needed if you want to download favorite creators or posts  
  
`--cookie PATH`  
cookie file, default is cookies.txt (value separate by whitespace)  
format:  

| Domain        | Include subdomains | Path | Secure | Expiry     | Name        | Value   |
|---------------|--------------------|------|--------|------------|-------------|---------|
| .kemono.party | FALSE              | /    | TRUE   | 1706755572 | kemono_auth | <value> |

you can get it easily by using Chrome extension [Get cookies.txt](https://chrome.google.com/webstore/detail/get-cookiestxt/bgaddhkoddajcdgocldbbfleckgcbcid)
#### Windows
windows can detect the cookie file automatically  
`--cookie-browser string`  
which browser to use, default is chrome (supported: chrome, firefox, edge , opera, vivaldi)

### Download Option
`--link [<urls>]`  
download link, separate by comma  
`--creator [<service>:<id>]`  
download creators, separate by comma  
`--banner bool`  
download banner, default is false (kemono only)  
`fav-site string`  
download fav-site (e.g. kemono, etc.), separate by comma.  
`--fav-creator bool`  
download favorite creator, default is false  
`--fav-post bool`  
download favorite post, default is false

### Post Filter Option
`--first int`  
download first n post    
`--last int`  
download last n post  
`--date YYYYMMDD`  
download post on date  
`--date-before YYYYMMDD`  
download post before date  
`--date-after YYYYMMDD`  
download post after date  
`--update YYYYMMDD`  
download post updated on date  
`--update-before YYYYMMDD`  
download post updated before date  
`--update-after YYYYMMDD`  
download post updated after date

### Image Filter Option
`--extensionOnly [<ext>]`  
download post with extension, separate by comma  
`--extensionExcept [<ext>]`  
download post without extension, separate by comma  
`--max-size string`  
download post with size less than max-size (e.g. 1 MB, 1KB, 1.5 gb, etc.)  
`--min-size string`  
download post with size greater than min-size (e.g. 1 MB, 1KB, 1.5 gb, etc.)

### Downloader Option
`--output PATH`  
output path  
`--overwrite bool`  
overwrite existing file  
`--async bool`  
download posts asynchronously, may cause the file order is not the same as the post order, can be used with --with-prefix-number, default false  
`--max-download-parallel int`  
max download file concurrent, default is 3, async mode only  
`--with-prefix-number bool`  
add prefix number to file name `<order>-<filename>`, default false  
`--name-rule-only-index bool`  
only use index as file name, default false  
`--download-timeout int`  
download timeout in seconds, default 300  
`--retry int`  
retry times, default 3  
`--retry-interval int`  
retry interval in seconds, default 10  
`--rate-limit int`  
rate limit in request/s, default 2

## Config File
config file is in `./config.yaml`  
Option in config is same as flag option, but without `--` prefix, and will be overridden by flag option .Can set the not often changed option in config file, and use flag option to override it  
Example:  
```yaml
banner: true
async: true
max-download-parallel: 5
with-prefix-number: true
```

## Features
With Kemono-scraper, you can implement a Downloader to take advantage of features such as multi-connection downloading, resume broken downloads, and more.

