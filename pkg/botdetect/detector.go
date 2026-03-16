package botdetect

import (
	"regexp"
	"strings"
)

var botPatterns = []string{
	"googlebot", "bingbot", "slurp", "duckduckbot", "baiduspider",
	"yandexbot", "sogou", "exabot", "facebot", "facebookexternalhit",
	"twitterbot", "linkedinbot", "whatsapp", "slackbot", "telegrambot",
	"discordbot", "applebot", "semrushbot", "ahrefsbot", "mj12bot",
	"dotbot", "petalbot", "bytespider", "gptbot", "claudebot",
	"ia_archiver", "archive.org_bot", "screaming frog",
}

type Detector struct {
	pattern *regexp.Regexp
}

func New() *Detector {
	combined := strings.Join(botPatterns, "|")
	return &Detector{
		pattern: regexp.MustCompile("(?i)(" + combined + ")"),
	}
}

func (d *Detector) IsBot(userAgent string) bool {
	return d.pattern.MatchString(userAgent)
}
