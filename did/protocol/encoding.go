package protocol

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	DomainDIDRegister    = "govdid:did-register:v1"
	DomainDIDUpdate      = "govdid:did-update:v1"
	DomainDIDDeactivate  = "govdid:did-deactivate:v1"
	DomainPolicyRegister = "govdid:policy-register:v1"
	DomainPolicyInactive = "govdid:policy-inactive:v1"
	DomainVC             = "govdid:vc:v1"
	DomainVP             = "govdid:vp:v1"
	DomainQualification  = "govdid:qualification:v1"
	DomainGroupAnchor    = "govdid:group-anchor:v1"
	DomainMessageContext = "govdid:message-context:v1"
	DomainLink           = "govdid:link-domain:v1"
	DomainRevCredential  = "govdid:revocation-credential:v1"
	DomainRevMessage     = "govdid:revocation-message:v1"
)

func EncodeFields(fields ...[]byte) []byte {
	total := 0
	for _, field := range fields {
		total += 4 + len(field)
	}
	out := make([]byte, 0, total)
	var size [4]byte
	for _, field := range fields {
		binary.BigEndian.PutUint32(size[:], uint32(len(field)))
		out = append(out, size[:]...)
		out = append(out, field...)
	}
	return out
}

func HashHex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func DecodeHash(value string) ([]byte, error) {
	value = strings.TrimPrefix(strings.TrimPrefix(value, "0x"), "0X")
	b, err := hex.DecodeString(value)
	if err != nil || len(b) != sha256.Size {
		return nil, errors.New("expected a 32-byte hex hash")
	}
	return b, nil
}

func NormalizePermission(permission Permission) (Permission, error) {
	permission.PolicyID = strings.TrimSpace(permission.PolicyID)
	permission.DeptRole = strings.TrimSpace(permission.DeptRole)
	permission.AuthScope = strings.TrimSpace(permission.AuthScope)
	permission.DataLevel = strings.TrimSpace(permission.DataLevel)
	if permission.PolicyID == "" || permission.DeptRole == "" || permission.AuthScope == "" || permission.DataLevel == "" {
		return Permission{}, errors.New("permission fields must not be empty")
	}
	if permission.ValidFrom < 0 || permission.ValidUntil <= permission.ValidFrom {
		return Permission{}, errors.New("invalid permission validity window")
	}
	actions := make([]string, 0, len(permission.ActionSet))
	seen := make(map[string]struct{}, len(permission.ActionSet))
	for _, action := range permission.ActionSet {
		action = strings.TrimSpace(action)
		if action == "" {
			return Permission{}, errors.New("permission action must not be empty")
		}
		if _, ok := seen[action]; ok {
			continue
		}
		seen[action] = struct{}{}
		actions = append(actions, action)
	}
	if len(actions) == 0 {
		return Permission{}, errors.New("permission action set must not be empty")
	}
	sort.Strings(actions)
	permission.ActionSet = actions
	return permission, nil
}

func PermissionBytes(permission Permission) ([]byte, error) {
	p, err := NormalizePermission(permission)
	if err != nil {
		return nil, err
	}
	actions := make([][]byte, 0, len(p.ActionSet))
	for _, action := range p.ActionSet {
		actions = append(actions, []byte(action))
	}
	return EncodeFields(
		[]byte(p.PolicyID), []byte(p.DeptRole), []byte(p.AuthScope), []byte(p.DataLevel),
		EncodeFields(actions...), int64Bytes(p.ValidFrom), int64Bytes(p.ValidUntil),
	), nil
}

func PolicyLeaf(permission Permission, action, issuerDID string) ([]byte, error) {
	p, err := NormalizePermission(permission)
	if err != nil {
		return nil, err
	}
	if !contains(p.ActionSet, action) {
		return nil, errors.New("action is not in permission action set")
	}
	path := EncodeFields([]byte(p.DeptRole), []byte(p.AuthScope), []byte(p.DataLevel), []byte(action))
	encoded := EncodeFields([]byte(p.PolicyID), path, int64Bytes(p.ValidFrom), int64Bytes(p.ValidUntil), []byte(issuerDID))
	leafInput := append([]byte{0x00}, encoded...)
	sum := sha256.Sum256(leafInput)
	return sum[:], nil
}

func VerifyMerkleProof(leaf []byte, proof []MerkleStep, rootHex string) error {
	if len(leaf) != sha256.Size {
		return errors.New("invalid leaf length")
	}
	current := append([]byte(nil), leaf...)
	for _, step := range proof {
		sibling, err := DecodeHash(step.Sibling)
		if err != nil {
			return fmt.Errorf("invalid sibling: %w", err)
		}
		var input []byte
		switch strings.ToUpper(step.Position) {
		case "L":
			input = append([]byte{0x01}, sibling...)
			input = append(input, current...)
		case "R":
			input = append([]byte{0x01}, current...)
			input = append(input, sibling...)
		default:
			return errors.New("proof position must be L or R")
		}
		h := sha256.Sum256(input)
		current = h[:]
	}
	root, err := DecodeHash(rootHex)
	if err != nil {
		return err
	}
	if !equalBytes(current, root) {
		return errors.New("Merkle proof does not match policy root")
	}
	return nil
}

func VerifyPermissionProofs(permission Permission, issuerDID, rootHex string, proofs []ActionProof) error {
	p, err := NormalizePermission(permission)
	if err != nil {
		return err
	}
	byAction := make(map[string][]MerkleStep, len(proofs))
	for _, proof := range proofs {
		if _, exists := byAction[proof.Action]; exists {
			return fmt.Errorf("duplicate proof for action %s", proof.Action)
		}
		byAction[proof.Action] = proof.Steps
	}
	if len(byAction) != len(p.ActionSet) {
		return errors.New("proof set must cover every action exactly once")
	}
	for _, action := range p.ActionSet {
		steps, ok := byAction[action]
		if !ok {
			return fmt.Errorf("missing proof for action %s", action)
		}
		leaf, err := PolicyLeaf(p, action, issuerDID)
		if err != nil {
			return err
		}
		if err := VerifyMerkleProof(leaf, steps, rootHex); err != nil {
			return fmt.Errorf("action %s: %w", action, err)
		}
	}
	return nil
}

func QualificationID(permission Permission, issuerDID string) (string, error) {
	b, err := PermissionBytes(permission)
	if err != nil {
		return "", err
	}
	return HashHex(EncodeFields([]byte(DomainQualification), []byte(permission.PolicyID), b, []byte(issuerDID))), nil
}

func GroupHash(publicKeys []string, epoch uint64) string {
	keys := append([]string(nil), publicKeys...)
	sort.Strings(keys)
	encodedKeys := make([][]byte, 0, len(keys))
	for _, key := range keys {
		encodedKeys = append(encodedKeys, []byte(strings.ToLower(key)))
	}
	return HashHex(EncodeFields([]byte(DomainGroupAnchor), EncodeFields(encodedKeys...), uint64Bytes(epoch)))
}

func MessageContextBytes(ctx MessageContext) []byte {
	return EncodeFields(
		[]byte(DomainMessageContext), []byte(ctx.Nonce), []byte(ctx.VerifierDID), []byte(ctx.GroupID),
		[]byte(ctx.PolicyID), []byte(ctx.QID), []byte(ctx.LHash), int64Bytes(ctx.TimeSlot), []byte(ctx.Purpose),
	)
}

func LinkDomainBytes(publicKeys []string, policyID, qID string, timeSlot int64) []byte {
	keys := append([]string(nil), publicKeys...)
	sort.Strings(keys)
	encodedKeys := make([][]byte, 0, len(keys))
	for _, key := range keys {
		encodedKeys = append(encodedKeys, []byte(strings.ToLower(key)))
	}
	return EncodeFields([]byte(DomainLink), EncodeFields(encodedKeys...), []byte(policyID), []byte(qID), int64Bytes(timeSlot))
}

func RegisterDIDMessage(req RegisterDIDRequest) []byte {
	return EncodeFields([]byte(DomainDIDRegister), []byte(req.DID), []byte(req.DocumentHash), []byte(req.OwnerAddress), []byte(strings.ToLower(req.SigningPublicKey)), []byte(strings.ToLower(req.LSAGPublicKey)))
}

func UpdateDIDMessage(req UpdateDIDRequest) []byte {
	return EncodeFields([]byte(DomainDIDUpdate), []byte(req.DID), []byte(req.DocumentHash), []byte(strings.ToLower(req.SigningPublicKey)), []byte(strings.ToLower(req.LSAGPublicKey)))
}

func DeactivateDIDMessage(did string) []byte {
	return EncodeFields([]byte(DomainDIDDeactivate), []byte(did))
}

func RegisterPolicyMessage(req RegisterPolicyRequest) []byte {
	return EncodeFields([]byte(DomainPolicyRegister), []byte(req.PolicyID), []byte(strings.ToLower(req.PolicyHash)), []byte(req.IssuerDID), int64Bytes(int64(req.RuleCount)))
}

func DeactivatePolicyMessage(policyID string) []byte {
	return EncodeFields([]byte(DomainPolicyInactive), []byte(policyID))
}

func CredentialBytes(vc VerifiableCredential) ([]byte, error) {
	p, err := PermissionBytes(vc.Permission)
	if err != nil {
		return nil, err
	}
	return EncodeFields([]byte(DomainVC), []byte(vc.ID), []byte(vc.HolderDID), []byte(vc.IssuerDID), []byte(vc.PolicyID), p, int64Bytes(vc.IssuedAt), int64Bytes(vc.ExpiresAt), boolBytes(vc.AnonymousEligible)), nil
}

func CredentialHash(vc VerifiableCredential) (string, error) {
	b, err := CredentialBytes(vc)
	if err != nil {
		return "", err
	}
	return HashHex(EncodeFields(b, []byte(strings.ToLower(vc.Signature)))), nil
}

func VPBytes(vp StandardVP) ([]byte, error) {
	hashes := make([][]byte, 0, len(vp.Credentials))
	for _, credential := range vp.Credentials {
		h, err := CredentialHash(credential)
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, []byte(h))
	}
	return EncodeFields([]byte(DomainVP), []byte(vp.HolderDID), []byte(vp.VerifierDID), []byte(vp.Nonce), []byte(vp.Purpose), EncodeFields(hashes...)), nil
}

func MatchPermission(req AccessRequest, permission Permission) bool {
	p, err := NormalizePermission(permission)
	if err != nil {
		return false
	}
	at := req.AtTime
	return req.DeptRole == p.DeptRole && req.AuthScope == p.AuthScope && req.DataLevel == p.DataLevel && contains(p.ActionSet, req.Action) && at >= p.ValidFrom && at <= p.ValidUntil
}

func RevocationCredentialBytes(credential RevocationCredential) []byte {
	return EncodeFields([]byte(DomainRevCredential), []byte(credential.ID), []byte(credential.IssuerDID), []byte(credential.SubjectDID), []byte(credential.TargetVCID), []byte(credential.EventType), int64Bytes(credential.EventTime), []byte(credential.ScopeType), []byte(credential.ScopeID))
}

func RevocationCredentialHash(credential RevocationCredential) string {
	return HashHex(EncodeFields(RevocationCredentialBytes(credential), []byte(strings.ToLower(credential.Signature))))
}

func RevocationMessage(credentialHash, groupID, holderDID, vcID, nonce string) []byte {
	return EncodeFields([]byte(DomainRevMessage), []byte(strings.ToLower(credentialHash)), []byte(groupID), []byte(holderDID), []byte(vcID), []byte(nonce))
}

func int64Bytes(value int64) []byte {
	var out [8]byte
	binary.BigEndian.PutUint64(out[:], uint64(value))
	return out[:]
}

func uint64Bytes(value uint64) []byte {
	var out [8]byte
	binary.BigEndian.PutUint64(out[:], value)
	return out[:]
}

func boolBytes(value bool) []byte {
	if value {
		return []byte{1}
	}
	return []byte{0}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
