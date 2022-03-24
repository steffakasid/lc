package internal

import "strings"

func contains(arr []string, key string) (bool, string) {
	subfilter := ""
	for _, entry := range arr {
		var filter string
		if strings.Contains(entry, ".") {
			filterParts := strings.SplitAfterN(entry, ".", 2)
			filter = strings.TrimRight(filterParts[0], ".")
			subfilter = filterParts[1]
		} else {
			filter = entry
		}
		if filter == key {
			return true, subfilter
		}
	}
	return false, subfilter
}
