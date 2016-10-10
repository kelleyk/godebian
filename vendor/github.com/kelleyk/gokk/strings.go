package gokk

import "strings"

func Partition(s, sep string) (string, string, string) {
	i := strings.Index(s, sep)
	if i == -1 {
		return "", "", s
	}
	return s[0:i], s[i : i+1], s[i+1:]
}

// func PartitionFunc(s string, f func(rune) bool) (string, string) {
// 	i := strings.IndexFunc(s, f)
// 	if i == -1 {
// 		return "", s
// 	}
// 	return s[0:i], s[i:]
// }

func PartitionLast(s, sep string) (string, string, string) {
	i := strings.LastIndex(s, sep)
	if i == -1 {
		return s, "", ""
	}
	return s[0:i], s[i : i+1], s[i+1:]
}

// func PartitionLastFunc(s string, f func(rune) bool) (string, string) {
// 	i := strings.LastIndexFunc(s, f)
// 	if i == -1 {
// 		return s, ""
// 	}
// 	return s[0:i], s[i:]
// }

func TakeWhile(s string, f func(rune) bool) (string, string) {
	for i, r := range s {
		if !f(r) {
			return s[0:i], s[i:]
		}
	}
	return s, ""
}
