package util

import (
	"regexp"
	"strings"
)

type PathFilterRegex struct {
	Regex   *regexp.Regexp
	Exclude bool
}

func ToPathRegex(path string) (PathFilterRegex, error) {
	regexStr := path
	exclude := false
	if strings.HasPrefix(path, "!") {
		regexStr = strings.TrimPrefix(regexStr, "!")
		exclude = true
	}
	regex, err := regexp.Compile(regexStr)
	if err != nil {
		return PathFilterRegex{}, err
	}
	return PathFilterRegex{
		Regex:   regex,
		Exclude: exclude,
	}, nil
}
