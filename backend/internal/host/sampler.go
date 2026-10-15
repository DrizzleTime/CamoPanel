package host

// New reads metrics from the local machine. Read failures are returned to the caller.
func New() Sampler {
	return newLinux()
}
