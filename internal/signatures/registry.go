package signatures

// Source IDs are intentionally simple strings to keep configuration stable.
const (
	SourceMITREATTACK = "mitre_attack"
	SourceNVD         = "nvd"
	SourceOSV         = "osv"
	SourceCISAKEV     = "cisa_kev"
	SourceExploitDB   = "exploit_db"
)

var knownSources = map[string]struct{}{
	SourceMITREATTACK: {},
	SourceNVD:         {},
	SourceOSV:         {},
	SourceCISAKEV:     {},
	SourceExploitDB:   {},
}

// DefaultSources returns the default set of public sources available for opt-in.
func DefaultSources() []string {
	return []string{
		SourceMITREATTACK,
		SourceNVD,
		SourceOSV,
		SourceCISAKEV,
		SourceExploitDB,
	}
}

// IsKnownSource returns true when the source is a built-in source.
func IsKnownSource(id string) bool {
	_, ok := knownSources[id]
	return ok
}
