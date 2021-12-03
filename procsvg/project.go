package main

type Project struct {
	// source svg icon dir relative to project file
	IconDir string `json:"icondir"`

	// size subdirs for icons with multiple sizes or levels of detail
	SizeDirs []string `json:"sizedirs"`

	// intermediate dir relative to project file
	IntermediateDir string `json:"intermediatedir"`

	// conversion precision
	Epsilon float64 `json:"epsilon"`

	// target relative to project file
	Target string `json:"target"`
}

var DefaultProject = Project{
	IconDir:         "icons",
	IntermediateDir: "intermediate",
	SizeDirs:        []string{"."},
	Epsilon:         1e-4,
	Target:          "icons.iconpk",
}
