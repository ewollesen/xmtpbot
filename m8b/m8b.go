// Copyright (C) 2020 Eric Wollesen <ericw at xmtp dot net>

package m8b

import "math/rand"

// https://en.wikipedia.org/wiki/Magic_8-Ball
var responses = []string{
	// affirmative
	"It is certain.",
	"It is decidedly so.",
	"Without a doubt.",
	"Yes – definitely.",
	"You may rely on it.",
	"As I see it, yes.",
	"Most likely.",
	"Outlook good.",
	"Yes.",
	"Signs point to yes.",
	// non-committal
	"Reply hazy, try again.",
	"Ask again later.",
	"Better not tell you now.",
	"Cannot predict now.",
	"Concentrate and ask again.",
	// negative
	"Don't count on it.",
	"My reply is no.",
	"My sources say no.",
	"Outlook not so good.",
	"Very doubtful.",
}

func Ask(cmd string) string {
	return responses[rand.Intn(len(responses))]
}
