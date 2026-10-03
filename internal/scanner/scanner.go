package scanner

import (
	"errors"

	"github.com/tasnimzotder/portman/internal/model"
)

var (
	ErrUnsupportedPlatform = errors.New("unsupported platform")
	ErrNotImplemented      = errors.New("not implemented yet")
)

type Scanner interface {
	ListListeners() ([]model.Listener, error)
	GetPort(port int) (*model.Listener, error)
	FindByPattern(pattern string) ([]model.Listener, error)
	ListByPID(pid int) ([]model.Listener, error)
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

func New(opts Options) (Scanner, error) { return newPlatformScanner(opts) }
