package sweetiebot

import "testing"

func checkString(s string, expected string, fn string, t *testing.T) {
	if s != expected {
		t.Errorf("%v == %v", fn, s)
	}
}
func checkInt(i int, expected int, fn string, t *testing.T) {
	if i != expected {
		t.Errorf("%v == %v", fn, i)
	}
}
