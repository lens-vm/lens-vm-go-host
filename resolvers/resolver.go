package resolvers

import (
	"context"
)

type Resolver interface {
	// Schema returns a string indicating
	// what kind of URI schema it resolves.
	// This returned value is matched against the
	// URI prefix "<scheme>://..."
	Scheme() string
	Resolve(ctx context.Context, target string) ([]byte, error)
}

// Resolver Types
// - File Get - file://
// * HTTP Get - http://
// * IPFS - ipfs://
// * WebAssembly Package Manager (wapm.io) - wapm://
