package utils

import (
	"testing"
)

func TestParseURL(t *testing.T) {
	url, err := ParseUrl("http://www.fake.host:123/random/path")
	verifyUrl(t, url, err)
	verifyUrlString(t, url, true, true, true, "http://www.fake.host:123/random/path")
}

func TestParseURLInvalid(t *testing.T) {
	_, err := ParseUrl("http://www.fake.host:wrongport/random/path")
	if err == nil {
		t.Fatal("The port is wrong so an error was expected")
	}
}

func TestParseURLMissingScheme(t *testing.T) {
	url, err := ParseUrl("www.fake.host:80/random/path")
	verifyUrl(t, url, err)
	verifyUrlString(t, url, true, true, false, "http://www.fake.host:80")

	url, err = ParseUrl("www.fake.host:443/random/path")
	verifyUrl(t, url, err)
	verifyUrlString(t, url, true, true, false, "https://www.fake.host:443")
}

func TestParseURLMissingPort(t *testing.T) {
	url, err := ParseUrl("http://www.fake.host/random/path")
	verifyUrl(t, url, err)
	verifyUrlString(t, url, true, true, false, "http://www.fake.host:80")

	url, err = ParseUrl("https://www.fake.host/random/path")
	verifyUrl(t, url, err)
	verifyUrlString(t, url, true, true, false, "https://www.fake.host:443")
}

func TestPrintURL(t *testing.T) {
	url, err := ParseUrl("tcp://www.fake.host:123/random/path")
	verifyUrl(t, url, err)
	verifyUrlString(t, url, false, true, true, "www.fake.host:123/random/path")
	verifyUrlString(t, url, true, false, true, "tcp://www.fake.host/random/path")
	verifyUrlString(t, url, true, true, false, "tcp://www.fake.host:123")
	verifyUrlString(t, url, false, false, false, "www.fake.host")
}

func verifyUrl(t *testing.T, url *Url, err error) {
	if err != nil {
		t.Fatal("unexpected error: " + err.Error())
	}

	if url == nil {
		t.Fatal("parsed URL object shouldn't be null")
	}
}

func verifyUrlString(t *testing.T, url *Url, scheme, port, path bool, expected string) {
	urlString := url.String(scheme, port, path, false)
	if urlString != expected {
		t.Fatalf("Wrong URL string. Expected [%v] got [%v]", expected, urlString)
	}
}
