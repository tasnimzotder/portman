package scanner

import (
	"runtime"

	"github.com/tasnimzotder/portman/internal/model"
)

type Scanner interface {
	ListListeners() ([]model.Listener, error)
	GetPort(port int) (*model.Listener, error)
	FindByPattern(pattern string) ([]model.Listener, error)
}

type Options struct {
	IncludeTCP   bool
	IncludeUDP   bool
	IncludeIPv6  bool
	ResolveNames bool
	FetchStats   bool
}

func DefaultOptions() Options {
	return Options{
		IncludeTCP:   true,
		IncludeUDP:   true,
		IncludeIPv6:  true,
		ResolveNames: false,
		FetchStats:   false,
	}
}

func New(opts Options) Scanner {
	switch runtime.GOOS {
	case "linux":
		// TODO: implement later
		panic("not implemented yet")
	case "darwin":
		return NewDarwinScanner(opts)
	default:
		panic("unsupported platform: " + runtime.GOOS)
	}
}
