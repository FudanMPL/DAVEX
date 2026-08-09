package govdid

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"

	"didcontract/protocol"
)

type PolicyEntry struct {
	Permission protocol.Permission    `json:"permission"`
	Proofs     []protocol.ActionProof `json:"proofs"`
}

type PolicyBundle struct {
	PolicyID  string        `json:"policyID"`
	IssuerDID string        `json:"issuerDID"`
	Root      string        `json:"root"`
	RuleCount int           `json:"ruleCount"`
	Entries   []PolicyEntry `json:"entries"`
}

type policyLeaf struct {
	entry  int
	action string
	value  []byte
}

func BuildPolicy(issuerDID string, permissions []protocol.Permission) (*PolicyBundle, error) {
	if issuerDID == "" || len(permissions) == 0 {
		return nil, errors.New("issuer DID and at least one permission are required")
	}
	normalized := make([]protocol.Permission, len(permissions))
	leaves := make([]policyLeaf, 0)
	policyID := ""
	for index, permission := range permissions {
		p, err := protocol.NormalizePermission(permission)
		if err != nil {
			return nil, fmt.Errorf("permission %d: %w", index, err)
		}
		if policyID == "" {
			policyID = p.PolicyID
		} else if p.PolicyID != policyID {
			return nil, errors.New("all permissions in one policy bundle must use the same policyID")
		}
		normalized[index] = p
		for _, action := range p.ActionSet {
			leaf, err := protocol.PolicyLeaf(p, action, issuerDID)
			if err != nil {
				return nil, err
			}
			leaves = append(leaves, policyLeaf{entry: index, action: action, value: leaf})
		}
	}
	sort.Slice(leaves, func(i, j int) bool { return bytes.Compare(leaves[i].value, leaves[j].value) < 0 })
	for i := 1; i < len(leaves); i++ {
		if bytes.Equal(leaves[i-1].value, leaves[i].value) {
			return nil, errors.New("policy contains a duplicate permission action leaf")
		}
	}
	proofs := make([][]protocol.MerkleStep, len(leaves))
	level := make([][]byte, len(leaves))
	positions := make([]int, len(leaves))
	for i := range leaves {
		level[i] = append([]byte(nil), leaves[i].value...)
		positions[i] = i
	}
	for len(level) > 1 {
		for leafIndex, position := range positions {
			sibling := position ^ 1
			if sibling >= len(level) {
				sibling = position
			}
			direction := "L"
			if position%2 == 0 {
				direction = "R"
			}
			proofs[leafIndex] = append(proofs[leafIndex], protocol.MerkleStep{Sibling: hex.EncodeToString(level[sibling]), Position: direction})
			positions[leafIndex] = position / 2
		}
		next := make([][]byte, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			right := i + 1
			if right >= len(level) {
				right = i
			}
			input := append([]byte{0x01}, level[i]...)
			input = append(input, level[right]...)
			hash := sha256.Sum256(input)
			next = append(next, hash[:])
		}
		level = next
	}
	entries := make([]PolicyEntry, len(normalized))
	for i, permission := range normalized {
		entries[i] = PolicyEntry{Permission: permission}
	}
	for i, leaf := range leaves {
		entries[leaf.entry].Proofs = append(entries[leaf.entry].Proofs, protocol.ActionProof{Action: leaf.action, Steps: proofs[i]})
	}
	for i := range entries {
		sort.Slice(entries[i].Proofs, func(a, b int) bool { return entries[i].Proofs[a].Action < entries[i].Proofs[b].Action })
	}
	bundle := &PolicyBundle{PolicyID: policyID, IssuerDID: issuerDID, Root: hex.EncodeToString(level[0]), RuleCount: len(leaves), Entries: entries}
	for _, entry := range bundle.Entries {
		if err := protocol.VerifyPermissionProofs(entry.Permission, issuerDID, bundle.Root, entry.Proofs); err != nil {
			return nil, fmt.Errorf("self-check generated policy proof: %w", err)
		}
	}
	return bundle, nil
}

func (b PolicyBundle) RegistrationRequest() protocol.RegisterPolicyRequest {
	return protocol.RegisterPolicyRequest{PolicyID: b.PolicyID, PolicyHash: b.Root, IssuerDID: b.IssuerDID, RuleCount: b.RuleCount}
}

func (b PolicyBundle) Entry(index int) (PolicyEntry, error) {
	if index < 0 || index >= len(b.Entries) {
		return PolicyEntry{}, fmt.Errorf("policy entry index %d is out of range", index)
	}
	return b.Entries[index], nil
}
