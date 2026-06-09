package settings

var validStyles = map[string]bool{
	"black_grey_realism":   true,
	"color_realism":        true,
	"micro_realism":        true,
	"fine_line":            true,
	"single_needle":        true,
	"minimalist":           true,
	"blackwork":            true,
	"american_traditional": true,
	"neo_traditional":      true,
	"japanese":             true,
	"tribal":               true,
	"dotwork":              true,
	"geometric":            true,
	"illustrative":         true,
	"watercolor":           true,
	"lettering":            true,
}

func allValidStyles(styles []string) bool {
	for _, s := range styles {
		if !validStyles[s] {
			return false
		}
	}
	return true
}
