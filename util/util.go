// Copyright (C) 2016 Eric Wollesen <ericw at xmtp dot net>

package util

import "strings"

func EscapeMarkdown(input string) string {
	input = strings.Replace(input, "_", "\\_", -1)
	input = strings.Replace(input, "*", "\\*", -1)

	return input
}
