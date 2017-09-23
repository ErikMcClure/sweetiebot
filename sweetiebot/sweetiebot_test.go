package sweetiebot

import "testing"

func TestVersion(t *testing.T) {
	t.Parallel()

	ver := &Version{0, 0, 0, 0}

	checkString(ver.String(), "0.0", "ver.String()", t)
	checkInt(ver.Integer(), 0, "ver.Integer()", t)

	ver = &Version{1, 2, 3, 4}

	checkString(ver.String(), "1.2.3.4", "ver.String()", t)
	checkInt(ver.Integer(), 16909060, "ver.Integer()", t)

	ver = &Version{1, 2, 3, 0}

	checkString(ver.String(), "1.2.3", "ver.String()", t)
	checkInt(ver.Integer(), 16909056, "ver.Integer()", t)

	ver = &Version{1, 2, 0, 0}

	checkString(ver.String(), "1.2", "ver.String()", t)
	checkInt(ver.Integer(), 16908288, "ver.Integer()", t)

	ver = &Version{1, 0, 0, 0}

	checkString(ver.String(), "1.0", "ver.String()", t)
	checkInt(ver.Integer(), 16777216, "ver.Integer()", t)
}
