package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mensfeld/claude-on-incus/internal/container"
	"github.com/mensfeld/claude-on-incus/internal/session"
)

// SessionFixture creates a session setup for testing
// Similar to FactoryBot factories in claude_yard
type SessionFixture struct {
	Workspace     string
	SessionID     string
	ContainerName string
	SessionsDir   string
	Slot          int
	Privileged    bool
	Persistent    bool
	Image         string
	StoragePath   string
	SSHKeyPath    string
	GitConfigPath string
}

// NewSessionFixture creates a basic session fixture
func NewSessionFixture(t *testing.T) *SessionFixture {
	t.Helper()

	workspace := CreateTestWorkspace(t)
	sessionsDir := t.TempDir()
	sessionID, err := session.GenerateSessionID()
	if err != nil {
		t.Fatalf("Failed to generate session ID: %v", err)
	}

	return &SessionFixture{
		Workspace:     workspace,
		SessionID:     sessionID,
		ContainerName: session.ContainerName(workspace, 1),
		SessionsDir:   sessionsDir,
		Slot:          1,
		Privileged:    false,
		Image:         session.SandboxImage,
	}
}

// WithSlot creates a fixture with specific slot (trait pattern)
func (f *SessionFixture) WithSlot(slot int) *SessionFixture {
	f.Slot = slot
	f.ContainerName = session.ContainerName(f.Workspace, slot)
	return f
}

// WithPrivileged creates a privileged session fixture (trait pattern)
func (f *SessionFixture) WithPrivileged() *SessionFixture {
	f.Privileged = true
	f.Image = session.PrivilegedImage
	return f
}

// WithImage sets a custom image (trait pattern)
func (f *SessionFixture) WithImage(image string) *SessionFixture {
	f.Image = image
	return f
}

// WithStorage adds storage path (trait pattern)
func (f *SessionFixture) WithStorage(path string) *SessionFixture {
	f.StoragePath = path
	return f
}

// WithSSHKey adds SSH key path (trait pattern)
func (f *SessionFixture) WithSSHKey(path string) *SessionFixture {
	f.SSHKeyPath = path
	return f
}

// WithGitConfig adds git config path (trait pattern)
func (f *SessionFixture) WithGitConfig(path string) *SessionFixture {
	f.GitConfigPath = path
	return f
}

// WithPersistent makes session persistent (trait pattern)
func (f *SessionFixture) WithPersistent() *SessionFixture {
	f.Persistent = true
	return f
}

// Setup runs session.Setup with fixture options
func (f *SessionFixture) Setup(t *testing.T) *session.SetupResult {
	t.Helper()

	opts := session.SetupOptions{
		WorkspacePath: f.Workspace,
		Image:         f.Image,
		Privileged:    f.Privileged,
		Persistent:    f.Persistent,
		Slot:          f.Slot,
		SessionsDir:   f.SessionsDir,
		StoragePath:   f.StoragePath,
		SSHKeyPath:    f.SSHKeyPath,
		GitConfigPath: f.GitConfigPath,
	}

	result, err := session.Setup(opts)
	if err != nil {
		t.Fatalf("Fixture setup failed: %v", err)
	}

	return result
}

// Cleanup runs session.Cleanup with fixture options
func (f *SessionFixture) Cleanup(t *testing.T, saveSession bool) error {
	t.Helper()

	opts := session.CleanupOptions{
		ContainerName: f.ContainerName,
		SessionID:     f.SessionID,
		SessionsDir:   f.SessionsDir,
		SaveSession:   saveSession,
		Privileged:    f.Privileged,
		Persistent:    f.Persistent,
	}

	return session.Cleanup(opts)
}

// ContainerFixture creates a simple container for testing
type ContainerFixture struct {
	Name      string
	Image     string
	Ephemeral bool
	Manager   *container.Manager
}

// NewContainerFixture creates a basic container fixture
func NewContainerFixture(t *testing.T, name string) *ContainerFixture {
	t.Helper()

	return &ContainerFixture{
		Name:      name,
		Image:     "images:ubuntu/22.04",
		Ephemeral: true,
		Manager:   container.NewManager(name),
	}
}

// WithImage sets a custom image (trait pattern)
func (f *ContainerFixture) WithImage(image string) *ContainerFixture {
	f.Image = image
	return f
}

// WithPersistent makes container persistent (trait pattern)
func (f *ContainerFixture) WithPersistent() *ContainerFixture {
	f.Ephemeral = false
	return f
}

// Launch launches the container
func (f *ContainerFixture) Launch(t *testing.T) error {
	t.Helper()
	return f.Manager.Launch(f.Image, f.Ephemeral)
}

// Cleanup stops and deletes the container
func (f *ContainerFixture) Cleanup(t *testing.T) {
	t.Helper()
	_ = f.Manager.Stop(true)
	_ = f.Manager.Delete(true)
}

// WorkspaceFixture creates a workspace with test files
type WorkspaceFixture struct {
	Path  string
	Files map[string]string // filename -> content
}

// NewWorkspaceFixture creates a workspace fixture
func NewWorkspaceFixture(t *testing.T) *WorkspaceFixture {
	t.Helper()

	return &WorkspaceFixture{
		Path:  t.TempDir(),
		Files: make(map[string]string),
	}
}

// WithFile adds a file to the workspace (trait pattern)
func (f *WorkspaceFixture) WithFile(name, content string) *WorkspaceFixture {
	f.Files[name] = content
	return f
}

// WithLargeFile adds a large file (trait pattern)
func (f *WorkspaceFixture) WithLargeFile(name string, sizeBytes int) *WorkspaceFixture {
	// Create content filled with 'A' characters
	content := make([]byte, sizeBytes)
	for i := range content {
		content[i] = 'A'
	}
	f.Files[name] = string(content)
	return f
}

// Create creates the workspace with all files
func (f *WorkspaceFixture) Create(t *testing.T) string {
	t.Helper()

	for name, content := range f.Files {
		path := filepath.Join(f.Path, name)
		dir := filepath.Dir(path)

		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	return f.Path
}
