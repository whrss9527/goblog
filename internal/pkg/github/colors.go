package github

import "strings"

// languageColors are the colours GitHub (linguist) uses for the languages a
// personal project is likely to be written in.
var languageColors = map[string]string{
	"c":                "#555555",
	"c#":               "#178600",
	"c++":              "#f34b7d",
	"css":              "#663399",
	"dart":             "#00b4ab",
	"dockerfile":       "#384d54",
	"go":               "#00add8",
	"html":             "#e34c26",
	"java":             "#b07219",
	"javascript":       "#f1e05a",
	"jupyter notebook": "#da5b0b",
	"kotlin":           "#a97bff",
	"lua":              "#000080",
	"makefile":         "#427819",
	"objective-c":      "#438eff",
	"php":              "#4f5d95",
	"python":           "#3572a5",
	"ruby":             "#701516",
	"rust":             "#dea584",
	"scss":             "#c6538c",
	"shell":            "#89e051",
	"svelte":           "#ff3e00",
	"swift":            "#f05138",
	"typescript":       "#3178c6",
	"vue":              "#41b883",
	"zig":              "#ec915c",
}

// LanguageColor returns the GitHub colour of a programming language, "" when
// it is not known.
func LanguageColor(language string) string {
	return languageColors[strings.ToLower(strings.TrimSpace(language))]
}
