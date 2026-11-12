// shell/update/memory.go

package update

import "github.com/smegg99/s99base/release/version"

// Memory records the ignored update and the version whose notes were handled.
type Memory struct {
	Ignored string `json:"ignored"`
	Notes   string `json:"notes"`
}

// Store persists one installation's update choices.
type Store interface {
	Load() (Memory, bool, error)
	Save(Memory) error
}

// Ignore persists a version without its leading v while retaining the notes state.
func (s *Service) Ignore(tag string) error {
	s.mu.Lock()
	err := s.loadLocked()
	if err == nil {
		next := s.memory
		next.Ignored = version.Trim(tag)
		err = s.saveLocked(next)
	}
	s.mu.Unlock()
	return s.report(err)
}

func (s *Service) loadLocked() error {
	if s.loaded {
		return nil
	}
	if s.opts.Store != nil {
		memory, ok, err := s.opts.Store.Load()
		if err != nil {
			return err
		}
		s.memory, s.stored = memory, ok
	}
	s.loaded = true
	return nil
}

func (s *Service) saveLocked(memory Memory) error {
	if s.opts.Store != nil {
		if err := s.opts.Store.Save(memory); err != nil {
			return err
		}
	}
	s.memory, s.stored = memory, true
	return nil
}
