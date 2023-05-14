package rio

// Defines commone mode
const (
	Debug   = "debug"
	Release = "release"
)

// ReleaseMode defines application mode
// This should be set on start up to avoid race condition
var ReleaseMode = Release

// Mode alias for env mode
type Mode string
