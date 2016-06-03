// Copyright (C) 2016 Eric Wollesen <ericw at xmtp dot net>

package html

import (
	"io"
	"net/http"

	"github.com/spacemonkeygo/spacelog"

	go_html "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (
	logger = spacelog.GetLogger()
)

func ParseTitle(r io.Reader) string {
	d := go_html.NewTokenizer(r)
	in_title := false
	title := ""

	for {
		token_type := d.Next()
		if token_type == go_html.ErrorToken {
			return ""
		}

		switch token_type {
		case go_html.StartTagToken:
			token := d.Token()
			if token.DataAtom == atom.Title {
				in_title = true
				logger.Debugf("token: %+v", token)
				continue
			}
		case go_html.TextToken:
			if in_title {
				token := d.Token()
				logger.Debugf("token: %+v", token)
				title += token.Data
				continue
			}
		case go_html.EndTagToken:
			if in_title {
				token := d.Token()
				logger.Debugf("token: %+v", token)
				return title
			}
		}

	}
}

func ParseTitleFromURL(url string) (title string, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return ParseTitle(resp.Body), nil
}
