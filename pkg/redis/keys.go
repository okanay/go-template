package redis

import (
	"fmt"
	"sort"
	"strings"
)

const (
	KeyPrefix = "app" // Proje prefixi (opsiyonel)
)

// BuildKeyItem -> app:blog:1
func BuildKeyItem(domain string, id string) string {
	return fmt.Sprintf("%s:%s:%s", KeyPrefix, domain, id)
}

// BuildKeyView -> app:blog:1:view:card
func BuildKeyView(domain string, id string, view string) string {
	return fmt.Sprintf("%s:%s:%s:view:%s", KeyPrefix, domain, id, view)
}

// BuildWildcardKey -> app:blog:1:* (Scan işlemleri için)
func BuildWildcardKey(domain string, id string) string {
	return fmt.Sprintf("%s:%s:%s:*", KeyPrefix, domain, id)
}

// BuildKeyDep -> app:deps:author:55 (Author 55'e bağımlı olanları tutar)
func BuildKeyDep(domain string, id string) string {
	return fmt.Sprintf("%s:deps:%s:%s", KeyPrefix, domain, id)
}

// BuildKeyList -> app:blog:list:page=1:sort=desc
func BuildKeyList(domain string, params map[string]string) string {
	if len(params) == 0 {
		return fmt.Sprintf("%s:%s:list:all", KeyPrefix, domain)
	}

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString(KeyPrefix)
	sb.WriteString(":")
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
