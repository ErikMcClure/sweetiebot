package sweetiebot

import (
	"testing"
)

func TestVersion(t *testing.T) {
	t.Parallel()

	ver := &Version{0, 0, 0, 0}

	Check(ver.String(), "0.0", t)
	Check(ver.Integer(), 0, t)

	ver = &Version{1, 2, 3, 4}

	Check(ver.String(), "1.2.3.4", t)
	Check(ver.Integer(), 16909060, t)

	ver = &Version{1, 2, 3, 0}

	Check(ver.String(), "1.2.3", t)
	Check(ver.Integer(), 16909056, t)

	ver = &Version{1, 2, 0, 0}

	Check(ver.String(), "1.2", t)
	Check(ver.Integer(), 16908288, t)

	ver = &Version{1, 0, 0, 0}

	Check(ver.String(), "1.0", t)
	Check(ver.Integer(), 16777216, t)
}
