package tailor

// WardrobeInfo for quick List and Show
type WardrobeInfo struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	// DisplayManagerNotice, when true, prints a localized note at the end of
	// wear explaining that the vendor ships LightDM by design and removed
	// other display managers, so the user does not mistake it for a bug.
	DisplayManagerNotice bool `yaml:"display_manager_notice"`
}

// Suit represents the index.yaml standard structure
type Suit struct {
	Name                 string        `yaml:"name"`
	Description          string        `yaml:"description"`
	Author               string        `yaml:"author"`
	Release              string        `yaml:"release"`
	Packages             []string      `yaml:"packages"`
	Accessories          []string      `yaml:"accessories"`
	Cmds                 []string      `yaml:"cmds"`
	Distributions        []string      `yaml:"distributions"`
	Sequence             *Sequence     `yaml:"sequence"`
	Finalize             *Finalize     `yaml:"finalize"`
	Reboot               bool          `yaml:"reboot"`
	DisplayManagerNotice bool          `yaml:"display_manager_notice"`
	PackagesNoRecommends []string      `yaml:"-"`
	// Populated by normalize() from Sequence.PackagesInteractive.
	// These packages are installed without DEBIAN_FRONTEND=noninteractive
	// so the user can respond to license prompts and debconf questions.
	PackagesInteractive []string `yaml:"-"`
}

// Sequence groups repositories, packages and accessories in nested form.
type Sequence struct {
	Repositories                *Repositories `yaml:"repositories"`
	Packages                    []string      `yaml:"packages"`
	PackagesNoInstallRecommends []string      `yaml:"packages_no_install_recommends"`
	PackagesInteractive         []string      `yaml:"packages_interactive"`
	Accessories                 []string      `yaml:"accessories"`
	Cmds                        []string      `yaml:"cmds"`
}

// Repositories describes apt source modifications before installation.
type Repositories struct {
	SourcesList  []string `yaml:"sources_list"`   // components to enable: main, contrib, non-free...
	SourcesListD []string `yaml:"sources_list_d"` // shell command strings (third-party repo setup)
	Update       bool     `yaml:"update"`
	Upgrade      bool     `yaml:"upgrade"`
}

// Finalize groups commands executed at the end of costume application in nested form.
type Finalize struct {
	Customize bool     `yaml:"customize"`
	Cmds      []string `yaml:"cmds"`
}

func (s *Suit) normalize() {
	if s.Sequence != nil {
		s.Packages = append(s.Packages, s.Sequence.Packages...)
		s.Accessories = append(s.Accessories, s.Sequence.Accessories...)
		s.Cmds = append(s.Cmds, s.Sequence.Cmds...)
		s.PackagesNoRecommends = append(s.PackagesNoRecommends, s.Sequence.PackagesNoInstallRecommends...)
		s.PackagesInteractive = append(s.PackagesInteractive, s.Sequence.PackagesInteractive...)
	}
	if s.Finalize != nil {
		s.Cmds = append(s.Cmds, s.Finalize.Cmds...)
	}
}
