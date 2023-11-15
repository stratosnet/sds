package utils

import (
	netUrl "net/url"
	"strings"
)

type Url struct {
	Scheme   string
	Host     string
	Port     string
	Path     string
	RawQuery string
}

func ParseUrl(url string) (*Url, error) {
	splitUrl := strings.Split(url, "://")
	missingScheme := false
	if len(splitUrl) == 1 {
		url = "http://" + url
		missingScheme = true
	}

	parsedNetUrl, err := netUrl.Parse(url)
	if err != nil {
		return nil, err
	}

	parsedUrl := &Url{
		Scheme:   parsedNetUrl.Scheme,
		Host:     parsedNetUrl.Host,
		Path:     parsedNetUrl.Path,
		RawQuery: parsedNetUrl.RawQuery,
	}

	if missingScheme {
		parsedUrl.Scheme = ""
	}

	splitHost := strings.Split(parsedUrl.Host, ":")
	if len(splitHost) == 2 {
		parsedUrl.Host = splitHost[0]
		parsedUrl.Port = splitHost[1]
	}

	// Default values
	if parsedUrl.Scheme == "" {
		if parsedUrl.Port == "443" {
			parsedUrl.Scheme = "https"
		} else {
			parsedUrl.Scheme = "http"
		}
	}

	if parsedUrl.Port == "" {
		if parsedUrl.Scheme == "https" {
			parsedUrl.Port = "443"
		} else {
			parsedUrl.Port = "80"
		}
	}

	return parsedUrl, nil
}

func (url *Url) String(withScheme, withPort, withPath, withRawQuery bool) string {
	urlString := ""
	if withScheme && url.Scheme != "" {
		urlString = url.Scheme + "://"
	}

	urlString = urlString + url.Host

	if withPort && url.Port != "" {
		urlString = urlString + ":" + url.Port
	}

	if withPath && url.Path != "" {
		separator := "/"
		if url.Path[0] == '/' {
			separator = ""
		}
		urlString = urlString + separator + url.Path
	}

	if withRawQuery && url.RawQuery != "" {
		separator := "?"
		if url.RawQuery[0] == '?' {
			separator = ""
		}
		urlString = urlString + separator + url.RawQuery
	}

	return urlString
}
