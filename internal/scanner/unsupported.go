//go:build !darwin

package scanner

import (
	"fmt"
	"runtime"
)

func newPlatformScanner(opts Options) (Scanner, error) {
	if runtime.GOOS == "linux" {
		return nil, ErrNotImplemented
	}
	return nil, fmt.Errorf("%w: %s", ErrUnsupportedPlatform, runtime.GOOS)
}
