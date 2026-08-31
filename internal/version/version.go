package version

const (
	Major = 1
	Minor = 0
	Patch = 0
)

func String() string {
	return string(rune('0'+Major)) + "." + string(rune('0'+Minor)) + "." + string(rune('0'+Patch))
}
