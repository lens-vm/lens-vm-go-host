package resolvers

type Resolver interface {
	// Schema returns a string indicating
	// what kind of URI schema it resolves.
	// This returned value is matched against the
	// URI prefix "<scheme>://..."
	Scheme() string
	Resolve(target string) ([]byte, error)
}
