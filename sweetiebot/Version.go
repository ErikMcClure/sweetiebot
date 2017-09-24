package sweetiebot

import "fmt"

// Version represents an app version using four sections
type Version struct {
	major    byte
	minor    byte
	revision byte
	build    byte
}

func (v *Version) String() string {
	if v.build > 0 {
		return fmt.Sprintf("%v.%v.%v.%v", v.major, v.minor, v.revision, v.build)
	}
	if v.revision > 0 {
		return fmt.Sprintf("%v.%v.%v", v.major, v.minor, v.revision)
	}
	return fmt.Sprintf("%v.%v", v.major, v.minor)
}

// Integer gets the integer representation of the version
func (v *Version) Integer() int {
	return AssembleVersion(v.major, v.minor, v.revision, v.build)
}

// AssembleVersion creates a version integer out of four bytes
func AssembleVersion(major byte, minor byte, revision byte, build byte) int {
	return int(build) | (int(revision) << 8) | (int(minor) << 16) | (int(major) << 24)
}
