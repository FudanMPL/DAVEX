package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"didcontract/backend/internal/domain"
	"didcontract/protocol"
	"didcontract/sdk-go/govdid"
)

type State struct {
	Identities  map[string]govdid.Identity            `json:"identities"`
	Documents   map[string]string                     `json:"documents"`
	Policies    map[string]govdid.PolicyBundle        `json:"policies"`
	Credentials map[string]protocol.IssueVCRequest    `json:"credentials"`
	Sessions    map[string]domain.VerificationSession `json:"sessions"`
	Drafts      map[string]domain.RevocationDraft     `json:"revocationDrafts"`
	Groups      map[string]bool                       `json:"groups"`
}

func newState() State {
	return State{
		Identities: make(map[string]govdid.Identity), Documents: make(map[string]string),
		Policies: make(map[string]govdid.PolicyBundle), Credentials: make(map[string]protocol.IssueVCRequest),
		Sessions: make(map[string]domain.VerificationSession), Drafts: make(map[string]domain.RevocationDraft),
		Groups: make(map[string]bool),
	}
}

type Repository struct {
	mu        sync.RWMutex
	statePath string
	auditPath string
	state     State
}

func Open(statePath, auditPath string) (*Repository, error) {
	repository := &Repository{statePath: statePath, auditPath: auditPath, state: newState()}
	data, err := os.ReadFile(statePath)
	if err == nil {
		if err := json.Unmarshal(data, &repository.state); err != nil {
			return nil, fmt.Errorf("decode backend state: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read backend state: %w", err)
	}
	repository.initializeMaps()
	return repository, nil
}

func (r *Repository) initializeMaps() {
	if r.state.Identities == nil {
		r.state.Identities = make(map[string]govdid.Identity)
	}
	if r.state.Documents == nil {
		r.state.Documents = make(map[string]string)
	}
	if r.state.Policies == nil {
		r.state.Policies = make(map[string]govdid.PolicyBundle)
	}
	if r.state.Credentials == nil {
		r.state.Credentials = make(map[string]protocol.IssueVCRequest)
	}
	if r.state.Sessions == nil {
		r.state.Sessions = make(map[string]domain.VerificationSession)
	}
	if r.state.Drafts == nil {
		r.state.Drafts = make(map[string]domain.RevocationDraft)
	}
	if r.state.Groups == nil {
		r.state.Groups = make(map[string]bool)
	}
}

func (r *Repository) mutate(operation func(*State) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	data, err := json.Marshal(r.state)
	if err != nil {
		return err
	}
	var backup State
	if err := json.Unmarshal(data, &backup); err != nil {
		return err
	}
	if err := operation(&r.state); err != nil {
		return err
	}
	if err := r.saveLocked(); err != nil {
		r.state = backup
		return err
	}
	return nil
}

func (r *Repository) saveLocked() error {
	directory := filepath.Dir(r.statePath)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".backend-state-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(r.state); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(name, r.statePath)
}

func (r *Repository) PutIdentity(identity govdid.Identity, document string) error {
	return r.mutate(func(state *State) error {
		if _, exists := state.Identities[identity.DID]; exists {
			return fmt.Errorf("identity %s already exists", identity.DID)
		}
		state.Identities[identity.DID] = identity
		state.Documents[identity.DID] = document
		return nil
	})
}

func (r *Repository) ReplaceIdentity(identity govdid.Identity, document string) error {
	return r.mutate(func(state *State) error {
		if _, exists := state.Identities[identity.DID]; !exists {
			return fmt.Errorf("identity %s does not exist", identity.DID)
		}
		state.Identities[identity.DID] = identity
		state.Documents[identity.DID] = document
		return nil
	})
}

func (r *Repository) Identity(did string) (govdid.Identity, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	identity, ok := r.state.Identities[did]
	if !ok {
		return govdid.Identity{}, "", fmt.Errorf("identity %s not found", did)
	}
	return identity, r.state.Documents[did], nil
}

func (r *Repository) IdentityDIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]string, 0, len(r.state.Identities))
	for value := range r.state.Identities {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func (r *Repository) PutPolicy(bundle govdid.PolicyBundle) error {
	return r.mutate(func(state *State) error { state.Policies[bundle.PolicyID] = bundle; return nil })
}

func (r *Repository) Policy(id string) (govdid.PolicyBundle, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.state.Policies[id]
	if !ok {
		return govdid.PolicyBundle{}, fmt.Errorf("policy %s not found", id)
	}
	return value, nil
}

func (r *Repository) PutCredential(request protocol.IssueVCRequest) error {
	return r.mutate(func(state *State) error { state.Credentials[request.Credential.ID] = request; return nil })
}

func (r *Repository) Credential(id string) (protocol.IssueVCRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.state.Credentials[id]
	if !ok {
		return protocol.IssueVCRequest{}, fmt.Errorf("credential %s not found", id)
	}
	return value, nil
}

func (r *Repository) CredentialIDsForHolder(holderDID string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]string, 0)
	for id, credential := range r.state.Credentials {
		if credential.Credential.HolderDID == holderDID {
			values = append(values, id)
		}
	}
	sort.Strings(values)
	return values
}

func (r *Repository) PutSession(session domain.VerificationSession) error {
	return r.mutate(func(state *State) error { state.Sessions[session.Nonce] = session; return nil })
}

func (r *Repository) PutGroup(id string) error {
	return r.mutate(func(state *State) error { state.Groups[id] = true; return nil })
}

func (r *Repository) PutDraft(draft domain.RevocationDraft) error {
	return r.mutate(func(state *State) error {
		if _, exists := state.Drafts[draft.ID]; exists {
			return fmt.Errorf("draft %s already exists", draft.ID)
		}
		state.Drafts[draft.ID] = draft
		return nil
	})
}

func (r *Repository) Draft(id string) (domain.RevocationDraft, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.state.Drafts[id]
	if !ok {
		return domain.RevocationDraft{}, fmt.Errorf("revocation draft %s not found", id)
	}
	if value.Approvals == nil {
		value.Approvals = make(map[string]protocol.PartialSignature)
	}
	return value, nil
}

func (r *Repository) UpdateDraft(id string, update func(*domain.RevocationDraft) error) (domain.RevocationDraft, error) {
	var result domain.RevocationDraft
	err := r.mutate(func(state *State) error {
		value, ok := state.Drafts[id]
		if !ok {
			return fmt.Errorf("revocation draft %s not found", id)
		}
		if value.Approvals == nil {
			value.Approvals = make(map[string]protocol.PartialSignature)
		}
		if err := update(&value); err != nil {
			return err
		}
		state.Drafts[id] = value
		result = value
		return nil
	})
	return result, err
}

func (r *Repository) AppendAudit(entry domain.AuditEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(r.auditPath), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(r.auditPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(entry)
}

func (r *Repository) Audits(limit int) ([]domain.AuditEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	file, err := os.Open(r.auditPath)
	if errors.Is(err, os.ErrNotExist) {
		return []domain.AuditEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	entries := make([]domain.AuditEntry, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var entry domain.AuditEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	return entries, nil
}
