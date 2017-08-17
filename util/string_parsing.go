package util

import (
	"strings"

	"github.com/pkg/errors"
)

func CsvStringToMap(str string) (res map[string][]string, err error) {
	var (
		values []string
		pair   []string
	)

	res = make(map[string][]string)

	values = strings.Split(str, ",")
	for _, value := range values {
		pair = strings.SplitN(value, "=", 2)
		if len(pair) != 2 {
			err = errors.Errorf(
				"malformed string - not separated by '=' - %s",
				value)
			return
		}

		slice, present := res[pair[0]]
		if !present {
			res[pair[0]] = []string{pair[1]}
		} else {
			res[pair[0]] = append(slice, pair[1])
		}
	}

	return
}
