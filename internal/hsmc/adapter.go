package hsmc

var supportedAdapters = []string{
	"none",
	"codex",
	"command",
}

// SupportedAdapters returns the built-in adapter names understood by the CLI.
func SupportedAdapters() []string {
	return append([]string(nil), supportedAdapters...)
}
