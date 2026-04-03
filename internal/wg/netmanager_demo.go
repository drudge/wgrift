package wg

// demoNetManager is a NetManager that silently succeeds for all operations.
// Exists() returns true so interfaces appear "running" in demo mode.
type demoNetManager struct{}

// NewDemoNetManager returns a NetManager that no-ops all operations.
func NewDemoNetManager() NetManager { return &demoNetManager{} }

func (m *demoNetManager) Create(name string) error              { return nil }
func (m *demoNetManager) Delete(name string) error              { return nil }
func (m *demoNetManager) SetAddress(name, address string) error { return nil }
func (m *demoNetManager) SetMTU(name string, mtu int) error     { return nil }
func (m *demoNetManager) SetUp(name string) error               { return nil }
func (m *demoNetManager) SetDown(name string) error             { return nil }
func (m *demoNetManager) Exists(name string) (bool, error)      { return true, nil }
func (m *demoNetManager) QuickUp(name string) error             { return nil }
func (m *demoNetManager) QuickDown(name string) error           { return nil }
func (m *demoNetManager) SyncConf(name, confData string) error  { return nil }
func (m *demoNetManager) SaveConf(name, confData string) error  { return nil }
