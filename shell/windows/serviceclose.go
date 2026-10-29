// shell/windows/serviceclose.go
package windows

import "context"

// Close starts the calling window's close handshake, as its close button
// does. Main is refused: its close button hides every window or quits.
func (s *Service) Close(ctx context.Context, owner string) error {
	return s.close(CallingName(ctx), owner)
}

func (s *Service) close(name, owner string) error {
	if err := s.owns(owner); err != nil {
		return err
	}
	if name == "main" {
		return ErrWindow
	}
	s.mu.Lock()
	busy := s.quitting || s.closeToken[name] != ""
	s.mu.Unlock()
	if busy {
		return ErrBusy
	}
	s.requestClose(name)
	return nil
}

func (s *Service) requestClose(name string) {
	s.cancelDrag(name, "")
	if !s.registry.CanClose(name) {
		s.registry.Emit(name, "window:blocked", s.options.BlockedKey)
		return
	}
	token := Name()
	s.mu.Lock()
	s.closeToken[name] = token
	if s.quitting {
		s.closeEpoch[name] = s.quitEpoch
	} else {
		s.closeEpoch[name] = 0
	}
	s.mu.Unlock()
	s.registry.Emit(name, "window:close", token)
}

// RequestQuit asks every frontend to flush layout and accepted edits.
// It is also application.Options.ShouldQuit's gate.
func RequestQuit(s *Service) bool {
	s.movement.Lock()
	defer s.movement.Unlock()
	s.cancelDrag("", "")
	s.mu.Lock()
	if s.quitReady {
		s.mu.Unlock()
		return true
	}
	if s.quitting {
		s.mu.Unlock()
		return false
	}
	s.mu.Unlock()
	list := s.registry.List()
	for _, window := range list {
		if !s.registry.CanClose(window.Name) {
			s.registry.Emit(window.Name, "window:blocked", s.options.BlockedKey)
			return false
		}
	}
	s.mu.Lock()
	s.quitting = true
	s.quitEpoch++
	s.quitWaiting = map[string]bool{}
	for _, window := range list {
		s.quitWaiting[window.Name] = true
	}
	s.mu.Unlock()
	for _, window := range list {
		s.requestClose(window.Name)
	}
	return false
}

// CloseReady is called only after the frontend has flushed and stopped edits.
func (s *Service) CloseReady(ctx context.Context, token string, ready bool) error {
	return s.closeReady(CallingName(ctx), token, ready)
}

func (s *Service) closeReady(name, token string, ready bool) (result error) {
	s.mu.Lock()
	if s.closeToken[name] != token {
		s.mu.Unlock()
		return ErrWindow
	}
	defer func() {
		if result != nil {
			s.resume()
		}
	}()
	s.closeToken[name] = "closing"
	epoch := s.closeEpoch[name]
	delete(s.closeEpoch, name)
	if epoch != 0 && (!s.quitting || epoch != s.quitEpoch) {
		delete(s.closeToken, name)
		s.mu.Unlock()
		return nil
	}
	if !ready {
		s.quitting = false
		s.quitWaiting = nil
		s.mu.Unlock()
		s.resume()
		return nil
	}
	quitting, closeOwner := s.quitting, s.owner
	s.mu.Unlock()
	keepError := s.keep()
	s.mu.Lock()
	if s.owner != closeOwner {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()
	if keepError != nil {
		s.mu.Lock()
		s.quitting = false
		s.quitWaiting = nil
		s.mu.Unlock()
		s.resume()
		return keepError
	}
	if quitting {
		s.mu.Lock()
		if !s.quitting || epoch != s.quitEpoch {
			s.mu.Unlock()
			return nil
		}
		delete(s.quitWaiting, name)
		done := len(s.quitWaiting) == 0
		if done {
			s.quitReady = true
		}
		s.mu.Unlock()
		if done {
			s.quit()
		}
		return nil
	}
	if !s.registry.CanClose(name) {
		return ErrBusy
	}
	s.mu.Lock()
	if s.owner != closeOwner {
		s.mu.Unlock()
		return nil
	}
	stop, win := s.stopping[name], s.native[name]
	if closeOwner != "" && s.options.Drop != nil {
		if err := s.options.Drop(closeOwner, name); err != nil {
			s.mu.Unlock()
			return err
		}
	}
	if err := s.registry.Remove(name); err != nil {
		s.mu.Unlock()
		return err
	}
	s.approved[name] = true
	delete(s.native, name)
	delete(s.stopping, name)
	s.mu.Unlock()
	// Removed before Track's final flush, so geometry cannot recreate the entry.
	if stop != nil {
		stop()
	}
	if win != nil {
		win.Close()
	}
	s.broadcast()
	return nil
}

func (s *Service) resume() {
	s.mu.Lock()
	for name, token := range s.closeToken {
		if token == "closing" {
			delete(s.closeToken, name)
		}
	}
	s.mu.Unlock()
	for _, window := range s.registry.List() {
		s.registry.Emit(window.Name, "window:resume", "")
	}
}
