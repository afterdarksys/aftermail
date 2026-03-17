package accounts

// Global configuration options for the AfterMail client
var (
	// RobustParsingEnabled toggles heuristic-based recovery techniques for broken MIME/headers
	// When true, the client-side parsers will attempt to salvage poorly formatted 
	// email boundaries and normalize broken carriage returns (\n vs \r\n) automatically.
	RobustParsingEnabled bool = false
)
