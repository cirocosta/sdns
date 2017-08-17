package main

import (
	"strings"
)

func CsvToDomainMapping(str string) (mapping map[string]*Domain, err error) {
	var (
		internalMapping = make(map[string][]string)
		mapping = make(map[string]*Domain)
		parts   = strings.Split(str, ",")
	)

	for _, part := range parts {
		// separete '='
		// take key and value
	}
}

func CsvStringToMap(str string) map[string][]string{
		pair := strings.Split(part, "=")
		if len(pair) != 2 {

		}

		oldValue, present := internalMapping[
		if !present {
			internalMapping[key] = []string{value}
		} else {
			internalMapping[key] = append(oldValue, value)
		}
}

