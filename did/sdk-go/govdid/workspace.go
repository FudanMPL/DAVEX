package govdid

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"didcontract/protocol"
)

type Workspace struct {
	Identities  map[string]Identity                `json:"identities"`
	Policies    map[string]PolicyBundle            `json:"policies"`
	Credentials map[string]protocol.IssueVCRequest `json:"credentials"`
}

func NewWorkspace() *Workspace {
	return &Workspace{
		Identities: make(map[string]Identity), Policies: make(map[string]PolicyBundle),
		Credentials: make(map[string]protocol.IssueVCRequest),
	}
}

func LoadWorkspace(path string) (*Workspace, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return NewWorkspace(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read local workspace: %w", err)
	}
	workspace := NewWorkspace()
	if err := json.Unmarshal(data, workspace); err != nil {
		return nil, fmt.Errorf("decode local workspace: %w", err)
	}
	if workspace.Identities == nil {
		workspace.Identities = make(map[string]Identity)
	}
	if workspace.Policies == nil {
		workspace.Policies = make(map[string]PolicyBundle)
	}
	if workspace.Credentials == nil {
		workspace.Credentials = make(map[string]protocol.IssueVCRequest)
	}
	return workspace, nil
}

func (w *Workspace) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create workspace directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".workspace-*")
	if err != nil {
		return fmt.Errorf("create temporary workspace: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(w); err != nil {
		temporary.Close()
		return fmt.Errorf("encode local workspace: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace local workspace: %w", err)
	}
	return nil
}

func (w *Workspace) Identity(did string) (Identity, error) {
	identity, ok := w.Identities[did]
	if !ok {
		return Identity{}, fmt.Errorf("local identity %s not found", did)
	}
	return identity, nil
}

func (w *Workspace) IdentityDIDs() []string {
	dids := make([]string, 0, len(w.Identities))
	for did := range w.Identities {
		dids = append(dids, did)
	}
	sort.Strings(dids)
	return dids
}
