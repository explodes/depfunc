package depfunc

import (
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func funcQName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func funcName(i interface{}) string {
	qname := funcQName(i)
	parts := strings.Split(qname, ".")
	return parts[len(parts)-1]
}

func assertOccursBefore(t *testing.T, before rune, after string, within string) bool {
	t.Helper()

	beforeIndex := indexOf(before, within)
	if beforeIndex == -1 {
		t.Errorf(`"%c" was not found in "%v"`, before, within)
		return false
	}

	ok := true
	for _, a := range after {
		afterIndex := indexOf(a, within)
		if afterIndex == -1 {
			t.Errorf(`"%c" was not before "%c" because "%c" was not found in "%v"`, before, a, a, within)
			ok = false
		} else if beforeIndex >= afterIndex {
			t.Errorf(`"%c" was not before "%c" in "%v"`, before, a, within)
			ok = false
		}
	}
	return ok
}

func indexOf(needle rune, haystack string) int {
	for i, s := range haystack {
		if s == needle {
			return i
		}
	}
	return -1
}
