package kvm

import "regexp"

// IsValidVMName validates VM name against libvirt naming rules
// Rules: start with letter/underscore, contain only [a-zA-Z0-9_-], max 64 chars
func IsValidVMName(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_-]*$`, name)
	return matched && len(name) <= 64
}
