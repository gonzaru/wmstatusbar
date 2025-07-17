package features

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

/* public api */

// Feature interface for each implemented module (audio, date…)
type Feature interface {
	// Name returns the name of the feature
	Name() string
	// Info retrieves the feature information
	Info(ctx context.Context) (string, error)
}

// ParseList splits comma-separated list into a slice of strings
func ParseList(raw string) []string {
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// Option configures registry execution
type Option func(*Options)

// Separator returns an Option that sets the separator string
func Separator(s string) Option {
	return func(o *Options) {
		o.sep = s
	}
}

// WithParallel returns an Option to enable or disable parallel execution
func WithParallel(b bool) Option {
	return func(o *Options) {
		o.parallel = b
	}
}

// factory creates a new Feature instance
type factory func() Feature

var registry = make(map[string]factory)

// adds a feature factory to the registry
func register(name string, f factory) {
	if _, dup := registry[name]; dup {
		panic("feature " + name + " duplicated in registry")
	}
	registry[name] = f
}

// Options holds configuration for a Registry
type Options struct {
	sep      string
	parallel bool
}

func defaultOptions() *Options {
	return &Options{
		sep:      " | ",
		parallel: true,
	}
}

// Registry manages active features and their settings
type Registry struct {
	feats    []Feature
	sep      string
	parallel bool
}

// NewRegistry creates a Registry for the given feature names
func NewRegistry(names []string, opts ...Option) (*Registry, error) {
	options := defaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	var feats []Feature
	for _, f := range names {
		create, ok := registry[f]
		if !ok {
			return nil, fmt.Errorf("unknown feature %q", f)
		}
		feats = append(feats, create())
	}
	return &Registry{
		feats:    feats,
		sep:      options.sep,
		parallel: options.parallel,
	}, nil
}

// Status retrieves Info from all registered features
func (r *Registry) Status(ctx context.Context) (string, error) {
	if r.parallel {
		return r.statusParallel(ctx)
	}
	return r.statusSequential(ctx)
}

func (r *Registry) statusSequential(ctx context.Context) (string, error) {
	var parts []string
	for _, f := range r.feats {
		s, err := f.Info(ctx)
		if err != nil {
			return "", fmt.Errorf("feature %q: %w", f.Name(), err)
		}
		if s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, r.sep), nil
}

func (r *Registry) statusParallel(parent context.Context) (string, error) {
	var (
		results = make([]string, len(r.feats))
		errOnce struct {
			err  error
			once sync.Once
		}
		wg sync.WaitGroup
	)

	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	wg.Add(len(r.feats))
	for i, f := range r.feats {
		go func(i int, f Feature) {
			defer wg.Done()

			s, err := f.Info(ctx)
			if err != nil {
				errOnce.once.Do(func() {
					errOnce.err = fmt.Errorf("feature %q: %w", f.Name(), err)
					cancel()
				})
				return
			}
			if s != "" {
				results[i] = s
			}
		}(i, f)
	}
	wg.Wait()

	if errOnce.err != nil {
		return "", errOnce.err
	}

	var parts []string
	for _, s := range results {
		if s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, r.sep), nil
}
