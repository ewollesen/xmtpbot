// Copyright (C) 2016 Eric Wollesen <ericw at xmtp dot net>

package util

import (
	"encoding/base64"
	"math/rand"
	"strings"
)

func EscapeMarkdown(input string) string {
	input = strings.Replace(input, "_", "\\_", -1)
	input = strings.Replace(input, "*", "\\*", -1)

	return input
}

func RandomState(bytes int) (state string, err error) {
	buf := make([]byte, bytes)
	_, err = rand.Read(buf)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(buf), nil
}
