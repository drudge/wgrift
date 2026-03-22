//go:build !linux

package wg

// NewNetManager returns a stub NetManager on non-Linux platforms.
func NewNetManager() NetManager {
	return &stubNetManager{}
}

type stubNetManager struct{}

func (m *stubNetManager) Create(name string) error            { return ErrNotSupported }
func (m *stubNetManager) Delete(name string) error            { return ErrNotSupported }
func (m *stubNetManager) SetAddress(name, address string) error { return ErrNotSupported }
func (m *stubNetManager) SetMTU(name string, mtu int) error   { return ErrNotSupported }
func (m *stubNetManager) SetUp(name string) error             { return ErrNotSupported }
func (m *stubNetManager) SetDown(name string) error           { return ErrNotSupported }
func (m *stubNetManager) Exists(name string) (bool, error)    { return false, ErrNotSupported }
