package redis

import (
	"fmt"
	"sort"
	"strings"
)

// 1. Liste Key Üretici: "blog:list:limit=10:page=1"
func (r *RedisClient) BuildKeyList(domain string, params map[string]string) string {
	if len(params) == 0 {
		return fmt.Sprintf("%s:list:all", domain)
	}

	// Parametreleri alfabetik sırala (Deterministik olması için şart)
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString(string(domain))
	sb.WriteString(":list")

	for _, key := range keys {
		sb.WriteString(":")
		sb.WriteString(key)
		sb.WriteString("=")
		sb.WriteString(params[key])
	}
	return sb.String()
}

// 2. BuildKeyItem Key Üretici: "blog:item:123"
func (r *RedisClient) BuildKeyItem(domain string, id string) string {
	return fmt.Sprintf("%s:item:%s", domain, id)
}

// 3. Dependency Key Üretici: "deps:user:123"
func (r *RedisClient) BuildKeyDep(domain string, id string) string {
	return fmt.Sprintf("deps:%s:%s", domain, id)
}
