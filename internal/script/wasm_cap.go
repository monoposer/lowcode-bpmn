package script

// ScriptCapability gates script_host imports for WASM ScriptTasks (Paca-style).
type ScriptCapability string

const (
	ScriptCapHTTPFetch ScriptCapability = "http_fetch"
	ScriptCapLog       ScriptCapability = "log"
)

// ScriptCapSet is a permission set for WASM script modules.
type ScriptCapSet map[ScriptCapability]struct{}

func ParseScriptCapabilities(list []string) ScriptCapSet {
	s := make(ScriptCapSet, len(list))
	for _, c := range list {
		s[ScriptCapability(c)] = struct{}{}
	}
	return s
}

func (s ScriptCapSet) Has(c ScriptCapability) bool {
	if s == nil {
		return false
	}
	_, ok := s[c]
	return ok
}

// DefaultScriptCaps enables safe observability only.
var DefaultScriptCaps = ScriptCapSet{
	ScriptCapLog: {},
}
