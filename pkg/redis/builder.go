package redis

import (
	"fmt"
	"sort"
	"strings"
)

func BuildKeyList(domain string, params map[string]string) string {
	if len(params) == 0 {
		return fmt.Sprintf("%s:list:all", domain)
	}

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString(domain)
	sb.WriteString(":list")

	for _, key := range keys {
		sb.WriteString(":")
		sb.WriteString(key)
		sb.WriteString("=")
		sb.WriteString(params[key])
	}
	return sb.String()
}

func BuildKeyDep(domain string, id string) string {
	return fmt.Sprintf("deps:%s:%s", domain, id)
}

func BuildKeyItem(domain string, id string) string {
	return fmt.Sprintf("%s:item:%s", domain, id)
}

func BuildKeyItemWithView(domain string, view string, identifier string) string {
	return fmt.Sprintf("%s:%s:item:%s", domain, view, identifier)
}

func BuildKeyItemWildCard(domain string, identifier string) string {
	return fmt.Sprintf("%s:*:item:%s", domain, identifier)
}
