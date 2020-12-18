// Copyleft 2020

package interview_accountapi

import (
	"fmt"
	url3 "net/url"
	"path"
	"testing"
)

func TestUrlUtils1(t *testing.T) {
	url := "hTTps://me:there@example.com/foo-bar/hello%20world?bazz=wazz&lol=hol#hello%20world"

	u, v, e := parseURL(url)
	if e != nil {
		t.Fatalf("Failed parsing URL: %s: %s", e.Error(), url)
	}

	u.Scheme = "htTP"
	u.Host = fmt.Sprintf("www.%s", u.Host)
	pass, s := u.User.Password()
	if s {
		u.User = url3.UserPassword("you", pass)
	} else {
		u.User = url3.User("you")
	}
	u.Path = "the/book/is/holy"
	v.Set("cheese", "holey")
	v.Set("lol", ":D")
	v.Del("bazz")

	url2 := assembleURL(u, v)
	urlx := "htTP://you:there@www.example.com/the/book/is/holy?cheese=holey&lol=%3AD#hello%20world"

	if url2 != urlx {
		t.Errorf("URLs does not match (built, expected): %s %s", url2, urlx)
	}
}

func TestUrlUtils2(t *testing.T) {
	url := "he/is/not/the/Messiah"

	u, v, e := parseURL(url)
	if e != nil {
		t.Fatalf("Failed parsing URL: %s: %s", e.Error(), url)
	}

	u.Path = path.Join(u.Path, "just-a-very")
	v.Set("boy", "Naughty")

	url2 := assembleURL(u, v)
	urlx := "he/is/not/the/Messiah/just-a-very?boy=Naughty"

	if url2 != urlx {
		t.Errorf("URLs does not match (built, expected): %s %s", url2, urlx)
	}
}
