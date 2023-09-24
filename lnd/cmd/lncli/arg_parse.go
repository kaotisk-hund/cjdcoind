package main

import (
	"regexp"
	"strconv"
	"time"

	"github.com/kaotisk-hund/cjdcoind/btcutil/er"
)

// reTimeRange matches systemd.time-like short negative timeranges, e.g. "-200s".
var reTimeRange = regexp.MustCompile(`^-\d{1,18}[s|m|h|d|w|M|y]$`)

// secondsPer allows translating s(seconds), m(minutes), h(ours), d(ays),
// w(eeks), M(onths) and y(ears) into corresponding seconds.
var secondsPer = map[string]int64{
	"s": 1,
	"m": 60,
	"h": 3600,
	"d": 86400,
	"w": 604800,
	"M": 2630016,  // 30.44 days
	"y": 31557600, // 365.25 days
}

// parseTime parses UNIX timestamps or short timeranges inspired by sytemd (when starting with "-"),
// e.g. "-1M" for one month (30.44 days) ago.
func parseTime(s string, base time.Time) (uint64, er.R) {
	if reTimeRange.MatchString(s) {
		last := len(s) - 1

		d, errr := strconv.ParseInt(s[1:last], 10, 64)
		if errr != nil {
			return uint64(0), er.E(errr)
		}

		mul := secondsPer[string(s[last])]
		return uint64(base.Unix() - d*mul), nil
	}

	i, e := strconv.ParseUint(s, 10, 64)
	return i, er.E(e)
}
