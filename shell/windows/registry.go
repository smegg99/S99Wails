// shell/windows/registry.go

// Package windows coordinates native surfaces and acknowledged view transfers.
// It stores no application data, documents or credentials.
package windows

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"sync"
)

var (
	ErrWindow   = errors.New("windows: unknown window")
	ErrTransfer = errors.New("windows: invalid transfer")
	ErrBusy     = errors.New("windows: a transfer is still in progress")
)

// Surface is the native behavior the registry needs, replaceable in tests.
type Surface interface {
	Name() string
	Show()
	Hide()
	Focus()
	Focused() bool
	Visible() bool
	Emit(string, any)
	Close()
}

// Info names a workspace window; Label is supplied by its own frontend.
type Info struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Ready bool   `json:"ready"`
}

// Offer stays Source's until Target acknowledges its view; Payload is transient and never logged or stored.
type Offer struct {
	ID      string          `json:"id"`
	Source  string          `json:"source"`
	Target  string          `json:"target"`
	Tab     string          `json:"tab"`
	Slot    int             `json:"slot"`
	Payload json.RawMessage `json:"payload"`
	Claimed bool            `json:"claimed"`
}

type entry struct {
	surface Surface
	info    Info
}

// Registry serializes movement independently of Wails' binding goroutines.
type Registry struct {
	mu      sync.Mutex
	entries map[string]*entry
	offers  map[string]Offer
}

func New() *Registry { return &Registry{entries: map[string]*entry{}, offers: map[string]Offer{}} }

// Name produces an unguessable native name, independent of a tab's display label.
func Name() string { return "tab-" + token() }
func token() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(value[:])
}

func (r *Registry) Add(surface Surface) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := surface.Name()
	if name == "" || r.entries[name] != nil {
		return ErrWindow
	}
	r.entries[name] = &entry{surface: surface, info: Info{Name: name}}
	return nil
}

// Ready also replays outstanding handshakes after a webview reload.
func (r *Registry) Ready(name, label string) ([]Offer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e := r.entries[name]
	if e == nil {
		return nil, ErrWindow
	}
	e.info.Ready, e.info.Label = true, label
	out := []Offer{}
	for _, offer := range r.offers {
		if offer.Source == name || offer.Target == name {
			out = append(out, clone(offer))
		}
	}
	return out, nil
}

func (r *Registry) List() []Info {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Info, 0, len(r.entries))
	for _, e := range r.entries {
		out = append(out, e.info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Reset withdraws all transfers and readiness when the workspace owner changes; surfaces stay until the owner restores.
func (r *Registry) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.offers = map[string]Offer{}
	for _, e := range r.entries {
		e.info.Ready = false
	}
}

func clone(offer Offer) Offer {
	offer.Payload = append(json.RawMessage(nil), offer.Payload...)
	return offer
}

func (r *Registry) Move(source, target, tab string, slot int, payload json.RawMessage) (Offer, error) {
	if source == target || tab == "" || !json.Valid(payload) {
		return Offer{}, ErrTransfer
	}
	r.mu.Lock()
	from, to := r.entries[source], r.entries[target]
	if from == nil || to == nil {
		r.mu.Unlock()
		return Offer{}, ErrWindow
	}
	for _, offer := range r.offers {
		if offer.Source == source && offer.Tab == tab {
			r.mu.Unlock()
			return Offer{}, ErrBusy
		}
	}
	offer := Offer{ID: token(), Source: source, Target: target, Tab: tab, Slot: slot, Payload: append(json.RawMessage(nil), payload...)}
	r.offers[offer.ID] = offer
	ready, surface := to.info.Ready, to.surface
	r.mu.Unlock()
	if ready {
		surface.Emit(eventTabArrive, clone(offer))
	}
	return clone(offer), nil
}

// Claim lets only the destination release the original view; repeats are harmless and a release missed in a reload is replayed.
func (r *Registry) Claim(target, id string) error {
	r.mu.Lock()
	offer, ok := r.offers[id]
	if !ok || offer.Target != target {
		r.mu.Unlock()
		return ErrTransfer
	}
	source := r.entries[offer.Source]
	if source == nil {
		r.mu.Unlock()
		return ErrWindow
	}
	offer.Claimed = true
	r.offers[id] = offer
	surface := source.surface
	r.mu.Unlock()
	surface.Emit(eventTabRelease, clone(offer))
	return nil
}

// Released forgets the payload only after the original view has been removed.
func (r *Registry) Released(source, id string) error {
	r.mu.Lock()
	offer, ok := r.offers[id]
	if !ok {
		r.mu.Unlock()
		return nil
	}
	if offer.Source != source || !offer.Claimed {
		r.mu.Unlock()
		return ErrTransfer
	}
	delete(r.offers, id)
	from, to := r.entries[offer.Source], r.entries[offer.Target]
	r.mu.Unlock()
	if from != nil {
		from.surface.Emit(eventTabDone, id)
	}
	if to != nil {
		to.surface.Emit(eventTabDone, id)
	}
	return nil
}

// Cancel keeps the original and withdraws a destination that has not claimed it.
func (r *Registry) Cancel(source, id string) error {
	r.mu.Lock()
	offer, ok := r.offers[id]
	if !ok || offer.Source != source || offer.Claimed {
		r.mu.Unlock()
		return ErrTransfer
	}
	delete(r.offers, id)
	target := r.entries[offer.Target]
	from := r.entries[offer.Source]
	r.mu.Unlock()
	if target != nil {
		target.surface.Emit(eventTabCancel, id)
	}
	if from != nil {
		from.surface.Emit(eventTabCancelled, clone(offer))
	}
	return nil
}

// canClose refuses to destroy either side of an unfinished transfer.
func (r *Registry) canClose(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, offer := range r.offers {
		if offer.Source == name || offer.Target == name {
			return false
		}
	}
	return r.entries[name] != nil
}

func (r *Registry) Remove(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, offer := range r.offers {
		if offer.Source == name || offer.Target == name {
			return ErrBusy
		}
	}
	if r.entries[name] == nil {
		return ErrWindow
	}
	delete(r.entries, name)
	return nil
}

func (r *Registry) surfaces() []Surface {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Surface, 0, len(r.entries))
	for _, e := range r.entries {
		out = append(out, e.surface)
	}
	return out
}

func (r *Registry) AnyFocused() bool {
	for _, s := range r.surfaces() {
		if s.Focused() {
			return true
		}
	}
	return false
}
func (r *Registry) ShowAll() {
	for _, s := range r.surfaces() {
		s.Show()
	}
	r.Focus("main")
}
func (r *Registry) hideAll() {
	for _, s := range r.surfaces() {
		s.Hide()
	}
}
func (r *Registry) Toggle() {
	for _, s := range r.surfaces() {
		if s.Visible() {
			r.hideAll()
			return
		}
	}
	r.ShowAll()
}
func (r *Registry) Focus(name string) {
	for _, s := range r.surfaces() {
		if s.Name() == name {
			s.Show()
			s.Focus()
			return
		}
	}
}
func (r *Registry) Emit(name, event string, payload any) {
	for _, s := range r.surfaces() {
		if s.Name() == name {
			s.Emit(event, payload)
			return
		}
	}
}
