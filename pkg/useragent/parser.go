package useragent

import (
	ua "github.com/mssola/useragent"
)

type Parser struct{}

func New() *Parser {
	return &Parser{}
}

type Result struct {
	Browser string
	OS      string
}

func (p *Parser) Parse(userAgent string) Result {
	agent := ua.New(userAgent)
	browser, _ := agent.Browser()
	return Result{
		Browser: browser,
		OS:      agent.OS(),
	}
}
