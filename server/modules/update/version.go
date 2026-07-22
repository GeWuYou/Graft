package update

import (
	"fmt"
	"strconv"
	"strings"
)

const semanticVersionPartCount = 3

// Version 是更新选择使用的严格 SemVer 子集；发布标签允许带一个 v 前缀。
type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
}

// ParseVersion 解析 Graft 支持的 SemVer 版本。build metadata 不参与发布选择。
func ParseVersion(raw string) (Version, error) {
	value := strings.TrimPrefix(strings.TrimSpace(raw), "v")
	if build, _, ok := strings.Cut(value, "+"); ok {
		value = build
	}
	base, prerelease, hasPrerelease := strings.Cut(value, "-")
	parts := strings.Split(base, ".")
	if len(parts) != semanticVersionPartCount {
		return Version{}, fmt.Errorf("invalid semantic version %q", raw)
	}
	parsed, err := parseVersionParts(parts, raw)
	if err != nil {
		return Version{}, err
	}
	parsed.Prerelease = prerelease
	if hasPrerelease && prerelease == "" {
		return Version{}, fmt.Errorf("invalid semantic version %q", raw)
	}
	return parsed, nil
}

func parseVersionParts(parts []string, raw string) (Version, error) {
	parsed := Version{}
	for index, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return Version{}, fmt.Errorf("invalid semantic version %q", raw)
		}
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return Version{}, fmt.Errorf("invalid semantic version %q", raw)
		}
		switch index {
		case 0:
			parsed.Major = number
		case 1:
			parsed.Minor = number
		case semanticVersionPartCount - 1:
			parsed.Patch = number
		}
	}
	return parsed, nil
}

// String 返回不带 v 前缀的规范版本。
func (v Version) String() string {
	if v.Prerelease == "" {
		return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	}
	return fmt.Sprintf("%d.%d.%d-%s", v.Major, v.Minor, v.Patch, v.Prerelease)
}

// IsPrerelease 表示该版本属于预发布通道。
func (v Version) IsPrerelease() bool { return v.Prerelease != "" }

// Compare 返回 v 与 other 的排序关系。
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		return compareInt(v.Major, other.Major)
	}
	if v.Minor != other.Minor {
		return compareInt(v.Minor, other.Minor)
	}
	if v.Patch != other.Patch {
		return compareInt(v.Patch, other.Patch)
	}
	if v.Prerelease == other.Prerelease {
		return 0
	}
	if v.Prerelease == "" {
		return 1
	}
	if other.Prerelease == "" {
		return -1
	}
	return comparePrerelease(v.Prerelease, other.Prerelease)
}

func compareInt(left, right int) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func comparePrerelease(left, right string) int {
	leftParts, rightParts := strings.Split(left, "."), strings.Split(right, ".")
	for index := 0; index < len(leftParts) && index < len(rightParts); index++ {
		if leftParts[index] == rightParts[index] {
			continue
		}
		leftNumber, leftNumeric := strconv.Atoi(leftParts[index])
		rightNumber, rightNumeric := strconv.Atoi(rightParts[index])
		if leftNumeric == nil && rightNumeric == nil {
			return compareInt(leftNumber, rightNumber)
		}
		if leftNumeric == nil {
			return -1
		}
		if rightNumeric == nil {
			return 1
		}
		if leftParts[index] < rightParts[index] {
			return -1
		}
		return 1
	}
	return compareInt(len(leftParts), len(rightParts))
}
