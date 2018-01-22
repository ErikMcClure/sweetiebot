package sweetiebot

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

type MockCall []interface{}
type MockHistory []MockCall
type Mock struct {
	history  MockHistory
	expected MockHistory
	t        *testing.T
	Disable  bool
}
type MockAny struct{}

func NewMock(t *testing.T) *Mock {
	return &Mock{MockHistory{}, MockHistory{}, t, false}
}

func (m *Mock) Input(args ...interface{}) {
	m.history = append(m.history, args)
	if !m.Disable && (len(m.history) > len(m.expected) || !m.history[len(m.history)-1].Compare(m.expected[len(m.history)-1])) {
		_, fn, line, _ := runtime.Caller(2)
		_, fn2, line2, _ := runtime.Caller(3)
		fmt.Printf("[%s:%d] [%s:%d] Did not expect ", filepath.Base(fn), line, filepath.Base(fn2), line2)
		fmt.Println(args...)
		m.t.Fail()
	}
}
func (m *Mock) Expect(args ...interface{}) {
	m.expected = append(m.expected, args)
}
func (m *Mock) Check() bool {
	return len(m.expected) == len(m.history)
}

func (r MockHistory) String() string {
	lines := make([]string, len(r), len(r))
	for k, v := range r {
		args := make([]string, len(v), len(v))
		for i, arg := range v {
			args[i] = fmt.Sprint(arg)
		}
		lines[k] = "[" + strings.Join(args, ", ") + "]"
	}
	return "[" + strings.Join(lines, ", ") + "]"
}

func (a MockCall) Compare(b MockCall) bool {
	if len(a) != len(b) {
		return false
	}
	any := reflect.TypeOf(MockAny{})

	for k := range a {
		ta := reflect.TypeOf(a[k])
		tb := reflect.TypeOf(b[k])
		if ta == any || tb == any {
			continue
		}
		if ta != tb {
			return false
		}
		switch ta.Kind() {
		case reflect.Func:
			if reflect.ValueOf(a[k]).Pointer() != reflect.ValueOf(b[k]).Pointer() {
				return false
			}
		default:
			if !reflect.DeepEqual(a[k], b[k]) {
				return false
			}
		}
	}
	return true
}
