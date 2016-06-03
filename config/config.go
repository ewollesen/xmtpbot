// Copyright (C) 2016 Eric Wollesen <ericw at xmtp dot net>

package config

import (
	"flag"
	"os"
)

var (
	Dir = flag.String("config_dir", os.ExpandEnv("$HOME/.xmtpbot"),
		"directory in which to store config and state")
)
