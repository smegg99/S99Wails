// shell/update/notes.go

package update

import (
	"github.com/smegg99/s99base/release/mdlocale"
	"github.com/smegg99/s99base/release/version"
)

// Notes contains bundled release notes resolved to one language.
type Notes struct {
	Version string `json:"version"`
	Text    string `json:"text"`
	Lang    string `json:"lang"`
}

// State describes the running version and any pending update or release notes.
type State struct {
	Current string   `json:"current"`
	Update  *Release `json:"update,omitempty"`
	Notes   *Notes   `json:"notes,omitempty"`
}

// State resolves pending notes in locale and reports storage errors through OnError.
func (s *Service) State(locale string) State {
	s.mu.Lock()
	state, err := s.stateLocked(locale)
	s.mu.Unlock()
	s.report(err)
	return state
}

func (s *Service) stateLocked(locale string) (State, error) {
	state := State{Current: s.opts.Version}
	if err := s.loadLocked(); err != nil {
		return state, err
	}
	if s.latest != nil && s.latest.Version != s.memory.Ignored {
		release := *s.latest
		state.Update = &release
	}
	if s.opts.Version == "" {
		return state, nil
	}
	existing := !s.stored && s.opts.Existing != nil && s.opts.Existing()
	if !s.stored && !existing {
		next := s.memory
		next.Notes = s.opts.Version
		return state, s.saveLocked(next)
	}
	if existing || version.Compare(s.opts.Version, s.memory.Notes) > 0 {
		text, lang := mdlocale.Split(s.opts.Changelog).Resolve(locale, "en")
		if text != "" {
			state.Notes = &Notes{Version: s.opts.Version, Text: text, Lang: lang}
		}
	}
	return state, nil
}

// NotesShown records the running version while preserving the ignored update.
func (s *Service) NotesShown() error {
	s.mu.Lock()
	err := s.loadLocked()
	if err == nil {
		next := s.memory
		next.Notes = s.opts.Version
		err = s.saveLocked(next)
	}
	s.mu.Unlock()
	return s.report(err)
}
