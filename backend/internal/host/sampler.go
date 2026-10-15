package host

// New returns the demo sampler unless the panel is attached to a real Docker engine.
// Host numbers then come from this machine. A read failure stays an error.
func New(mode string) Sampler {
	if mode == "docker" {
		return newLinux()
	}
	return newDemo()
}
