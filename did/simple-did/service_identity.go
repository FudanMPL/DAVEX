package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"chainmaker.org/chainmaker/contract-sdk-go/v2/sdk"
	"didcontract/protocol"
)

const (
	rolePolicyManager    = "policy_manager"
	roleCredentialIssuer = "credential_issuer"
	roleVerifier         = "verifier"
)

var allowedRoles = map[string]struct{}{
	rolePolicyManager: {}, roleCredentialIssuer: {}, roleVerifier: {},
}

type Service struct {
	store *Store
}

func NewService() *Service {
	return &Service{store: &Store{}}
}

func validateID(name, value string) error {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maxIDLength {
		return fmt.Errorf("%s must contain 1..%d characters", name, maxIDLength)
	}
	return nil
}

func (s *Service) getSystemConfig() (*SystemConfig, error) {
	var config SystemConfig
	if err := s.store.getJSON(prefixSystem, "config", &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (s *Service) getDID(did string) (*protocol.DIDRegistry, error) {
	var registry protocol.DIDRegistry
	if err := s.store.getJSON(prefixDID, did, &registry); err != nil {
		if errors.Is(err, errNotFound) {
			return nil, errors.New("DID not found")
		}
		return nil, err
	}
	return &registry, nil
}

func (s *Service) requireActiveDID(did string) (*protocol.DIDRegistry, error) {
	registry, err := s.getDID(did)
	if err != nil {
		return nil, err
	}
	if registry.Status != protocol.StatusActive {
		return nil, errors.New("DID is not active")
	}
	return registry, nil
}

func (s *Service) requireSenderControls(did string) (*protocol.DIDRegistry, error) {
	registry, err := s.requireActiveDID(did)
	if err != nil {
		return nil, err
	}
	sender, err := senderAddress()
	if err != nil {
		return nil, err
	}
	if sender != registry.OwnerAddress {
		return nil, errors.New("transaction sender does not control DID")
	}
	return registry, nil
}

func (s *Service) isGovernanceDID(did string) bool {
	config, err := s.getSystemConfig()
	return err == nil && config.GovernanceDID != "" && config.GovernanceDID == did
}

func (s *Service) requireGovernanceSender() (string, error) {
	config, err := s.getSystemConfig()
	if err != nil {
		return "", err
	}
	if config.GovernanceDID == "" {
		return "", errors.New("governance DID has not been bound")
	}
	if _, err := s.requireSenderControls(config.GovernanceDID); err != nil {
		return "", err
	}
	return config.GovernanceDID, nil
}

func (s *Service) requireRole(did, role string) error {
	if s.isGovernanceDID(did) {
		return nil
	}
	var binding protocol.RoleBinding
	if err := s.store.getJSON(prefixRole, did, &binding); err != nil {
		return errors.New("DID has no system role")
	}
	for _, assigned := range binding.Roles {
		if assigned == role {
			return nil
		}
	}
	return fmt.Errorf("DID lacks required role %s", role)
}

func (s *Service) RegisterDID(req protocol.RegisterDIDRequest) (*protocol.DIDRegistry, error) {
	if err := validateID("DID", req.DID); err != nil || !strings.HasPrefix(req.DID, "did:") {
		return nil, errors.New("invalid DID")
	}
	if err := validateID("owner address", req.OwnerAddress); err != nil {
		return nil, err
	}
	if _, err := protocol.DecodeHash(req.DocumentHash); err != nil {
		return nil, fmt.Errorf("document hash: %w", err)
	}
	if err := validateP256PublicKey(req.SigningPublicKey); err != nil {
		return nil, err
	}
	if err := validateLSAGPoint(req.LSAGPublicKey); err != nil {
		return nil, err
	}
	sender, err := senderAddress()
	if err != nil {
		return nil, err
	}
	if sender != req.OwnerAddress {
		return nil, errors.New("owner address must equal transaction origin")
	}
	if exists, err := s.store.exists(prefixDID, req.DID); err != nil || exists {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("DID already exists")
	}
	if _, err := s.store.getString(prefixAddress, req.OwnerAddress); err == nil {
		return nil, errors.New("owner address is already bound to a DID")
	} else if !errors.Is(err, errNotFound) {
		return nil, err
	}
	if err := verifyECDSAP256(req.SigningPublicKey, req.Proof, protocol.RegisterDIDMessage(req)); err != nil {
		return nil, fmt.Errorf("DID self-signature: %w", err)
	}
	now, err := txTime()
	if err != nil {
		return nil, err
	}
	registry := &protocol.DIDRegistry{
		DID: req.DID, DocumentHash: normalizeHex(req.DocumentHash), OwnerAddress: req.OwnerAddress,
		SigningPublicKey: normalizeHex(req.SigningPublicKey), LSAGPublicKey: normalizeHex(req.LSAGPublicKey),
		Status: protocol.StatusActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.store.putJSON(prefixDID, req.DID, registry); err != nil {
		return nil, err
	}
	if err := s.store.putString(prefixAddress, req.OwnerAddress, req.DID); err != nil {
		return nil, err
	}
	sdk.Instance.EmitEvent(topicDIDRegistered, []string{req.DID, req.OwnerAddress})
	return registry, nil
}

func (s *Service) UpdateDID(req protocol.UpdateDIDRequest) (*protocol.DIDRegistry, error) {
	registry, err := s.requireSenderControls(req.DID)
	if err != nil {
		return nil, err
	}
	if _, err := protocol.DecodeHash(req.DocumentHash); err != nil {
		return nil, fmt.Errorf("document hash: %w", err)
	}
	if err := validateP256PublicKey(req.SigningPublicKey); err != nil {
		return nil, err
	}
	if err := validateLSAGPoint(req.LSAGPublicKey); err != nil {
		return nil, err
	}
	message := protocol.UpdateDIDMessage(req)
	if err := verifyECDSAP256(registry.SigningPublicKey, req.CurrentKeyProof, message); err != nil {
		return nil, fmt.Errorf("current DID key proof: %w", err)
	}
	if err := verifyECDSAP256(req.SigningPublicKey, req.NewKeyProof, message); err != nil {
		return nil, fmt.Errorf("new DID key proof: %w", err)
	}
	now, err := txTime()
	if err != nil {
		return nil, err
	}
	registry.DocumentHash = normalizeHex(req.DocumentHash)
	registry.SigningPublicKey = normalizeHex(req.SigningPublicKey)
	registry.LSAGPublicKey = normalizeHex(req.LSAGPublicKey)
	registry.UpdatedAt = now
	if err := s.store.putJSON(prefixDID, req.DID, registry); err != nil {
		return nil, err
	}
	sdk.Instance.EmitEvent(topicDIDUpdated, []string{req.DID})
	return registry, nil
}

func (s *Service) DeactivateDID(req protocol.DeactivateDIDRequest) error {
	registry, err := s.requireSenderControls(req.DID)
	if err != nil {
		return err
	}
	if s.isGovernanceDID(req.DID) {
		return errors.New("governance DID cannot be deactivated")
	}
	if err := verifyECDSAP256(registry.SigningPublicKey, req.Proof, protocol.DeactivateDIDMessage(req.DID)); err != nil {
		return err
	}
	now, err := txTime()
	if err != nil {
		return err
	}
	registry.Status = protocol.StatusInactive
	registry.UpdatedAt = now
	if err := s.store.putJSON(prefixDID, req.DID, registry); err != nil {
		return err
	}
	sdk.Instance.EmitEvent(topicDIDDeactivated, []string{req.DID})
	return nil
}

func (s *Service) SetGovernanceDID(req protocol.SetGovernanceDIDRequest) error {
	config, err := s.getSystemConfig()
	if err != nil {
		return err
	}
	if config.GovernanceDID != "" {
		return errors.New("governance DID is immutable once bound")
	}
	registry, err := s.requireActiveDID(req.DID)
	if err != nil {
		return err
	}
	sender, err := senderAddress()
	if err != nil {
		return err
	}
	if sender != config.BootstrapAddress || registry.OwnerAddress != sender {
		return errors.New("only bootstrap origin may bind its governance DID")
	}
	config.GovernanceDID = req.DID
	if err := s.store.putJSON(prefixSystem, "config", config); err != nil {
		return err
	}
	sdk.Instance.EmitEvent(topicGovernanceBound, []string{req.DID})
	return nil
}

func (s *Service) UpdateRoles(req protocol.UpdateRolesRequest) (*protocol.RoleBinding, error) {
	if _, err := s.requireGovernanceSender(); err != nil {
		return nil, err
	}
	if _, err := s.requireActiveDID(req.DID); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(req.Roles))
	roles := make([]string, 0, len(req.Roles))
	for _, role := range req.Roles {
		if _, ok := allowedRoles[role]; !ok {
			return nil, fmt.Errorf("unsupported role %s", role)
		}
		if _, ok := seen[role]; !ok {
			seen[role] = struct{}{}
			roles = append(roles, role)
		}
	}
	sort.Strings(roles)
	now, err := txTime()
	if err != nil {
		return nil, err
	}
	binding := &protocol.RoleBinding{DID: req.DID, Roles: roles, UpdatedAt: now}
	if err := s.store.putJSON(prefixRole, req.DID, binding); err != nil {
		return nil, err
	}
	return binding, nil
}

func (s *Service) RegisterPolicy(req protocol.RegisterPolicyRequest) (*protocol.PolicyAnchor, error) {
	issuer, err := s.requireSenderControls(req.IssuerDID)
	if err != nil {
		return nil, err
	}
	if err := s.requireRole(req.IssuerDID, rolePolicyManager); err != nil {
		return nil, err
	}
	if err := validateID("policy ID", req.PolicyID); err != nil || req.RuleCount <= 0 {
		return nil, errors.New("invalid policy ID or rule count")
	}
	if _, err := protocol.DecodeHash(req.PolicyHash); err != nil {
		return nil, fmt.Errorf("policy hash: %w", err)
	}
	if exists, err := s.store.exists(prefixPolicy, req.PolicyID); err != nil || exists {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("policy ID already exists and cannot be overwritten")
	}
	if err := verifyECDSAP256(issuer.SigningPublicKey, req.Signature, protocol.RegisterPolicyMessage(req)); err != nil {
		return nil, err
	}
	now, err := txTime()
	if err != nil {
		return nil, err
	}
	anchor := &protocol.PolicyAnchor{PolicyID: req.PolicyID, PolicyHash: normalizeHex(req.PolicyHash), IssuerDID: req.IssuerDID, RuleCount: req.RuleCount, Status: protocol.StatusActive, CreatedAt: now, UpdatedAt: now, Signature: normalizeHex(req.Signature)}
	if err := s.store.putJSON(prefixPolicy, req.PolicyID, anchor); err != nil {
		return nil, err
	}
	sdk.Instance.EmitEvent(topicPolicyRegistered, []string{req.PolicyID, req.IssuerDID})
	return anchor, nil
}

func (s *Service) DeactivatePolicy(req protocol.DeactivatePolicyRequest) (*protocol.PolicyAnchor, error) {
	anchor, err := s.getPolicy(req.PolicyID)
	if err != nil {
		return nil, err
	}
	issuer, err := s.requireSenderControls(anchor.IssuerDID)
	if err != nil {
		return nil, err
	}
	if err := s.requireRole(anchor.IssuerDID, rolePolicyManager); err != nil {
		return nil, err
	}
	if anchor.Status != protocol.StatusActive {
		return nil, errors.New("policy is not active")
	}
	if err := verifyECDSAP256(issuer.SigningPublicKey, req.Signature, protocol.DeactivatePolicyMessage(req.PolicyID)); err != nil {
		return nil, err
	}
	now, err := txTime()
	if err != nil {
		return nil, err
	}
	anchor.Status = protocol.StatusInactive
	anchor.UpdatedAt = now
	if err := s.store.putJSON(prefixPolicy, req.PolicyID, anchor); err != nil {
		return nil, err
	}
	sdk.Instance.EmitEvent(topicPolicyDeactivated, []string{req.PolicyID})
	return anchor, nil
}

func (s *Service) getPolicy(policyID string) (*protocol.PolicyAnchor, error) {
	var anchor protocol.PolicyAnchor
	if err := s.store.getJSON(prefixPolicy, policyID, &anchor); err != nil {
		if errors.Is(err, errNotFound) {
			return nil, errors.New("policy not found")
		}
		return nil, err
	}
	return &anchor, nil
}

func (s *Service) VerifyPolicyMembership(req protocol.VerifyPolicyMembershipRequest) error {
	anchor, err := s.getPolicy(req.PolicyID)
	if err != nil {
		return err
	}
	if anchor.Status != protocol.StatusActive || req.IssuerDID != anchor.IssuerDID || req.Permission.PolicyID != req.PolicyID {
		return errors.New("policy membership context mismatch")
	}
	if len(req.Proof) > maxProofDepth {
		return errors.New("Merkle proof exceeds maximum depth")
	}
	leaf, err := protocol.PolicyLeaf(req.Permission, req.Action, req.IssuerDID)
	if err != nil {
		return err
	}
	return protocol.VerifyMerkleProof(leaf, req.Proof, anchor.PolicyHash)
}

func (s *Service) validateCredential(vc protocol.VerifiableCredential, proofs []protocol.ActionProof, now int64) (*protocol.VCProof, error) {
	stored, err := s.getVCProof(vc.ID)
	if err != nil {
		return nil, err
	}
	hash, err := protocol.CredentialHash(vc)
	if err != nil || hash != stored.CredentialHash {
		return nil, errors.New("credential hash does not match VCProof")
	}
	if stored.Status != protocol.StatusValid || now < stored.IssuedAt || now > stored.ExpiresAt {
		return nil, errors.New("credential is not currently valid")
	}
	if vc.HolderDID != stored.HolderDID || vc.IssuerDID != stored.IssuerDID || vc.PolicyID != stored.PolicyID || vc.Permission.PolicyID != vc.PolicyID {
		return nil, errors.New("credential subject or policy binding mismatch")
	}
	issuer, err := s.requireActiveDID(vc.IssuerDID)
	if err != nil {
		return nil, err
	}
	if err := s.requireRole(vc.IssuerDID, roleCredentialIssuer); err != nil {
		return nil, err
	}
	if _, err := s.requireActiveDID(vc.HolderDID); err != nil {
		return nil, err
	}
	if err := verifyECDSAP256(issuer.SigningPublicKey, vc.Signature, mustCredentialBytes(vc)); err != nil {
		return nil, fmt.Errorf("credential signature: %w", err)
	}
	anchor, err := s.getPolicy(vc.PolicyID)
	if err != nil || anchor.Status != protocol.StatusActive || anchor.IssuerDID != vc.IssuerDID {
		return nil, errors.New("credential policy anchor is not active or issuer-bound")
	}
	if err := protocol.VerifyPermissionProofs(vc.Permission, vc.IssuerDID, anchor.PolicyHash, proofs); err != nil {
		return nil, err
	}
	return stored, nil
}

func mustCredentialBytes(vc protocol.VerifiableCredential) []byte {
	b, _ := protocol.CredentialBytes(vc)
	return b
}

func (s *Service) IssueVC(req protocol.IssueVCRequest) (*protocol.VCProof, error) {
	vc := req.Credential
	issuer, err := s.requireSenderControls(vc.IssuerDID)
	if err != nil {
		return nil, err
	}
	if err := s.requireRole(vc.IssuerDID, roleCredentialIssuer); err != nil {
		return nil, err
	}
	if err := validateID("VC ID", vc.ID); err != nil || vc.PolicyID != vc.Permission.PolicyID {
		return nil, errors.New("invalid VC or policy binding")
	}
	if exists, err := s.store.exists(prefixVC, vc.ID); err != nil || exists {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("VC ID already exists")
	}
	if _, err := s.requireActiveDID(vc.HolderDID); err != nil {
		return nil, err
	}
	anchor, err := s.getPolicy(vc.PolicyID)
	if err != nil || anchor.Status != protocol.StatusActive || anchor.IssuerDID != vc.IssuerDID {
		return nil, errors.New("VC issuer does not own an active policy anchor")
	}
	now, err := txTime()
	if err != nil {
		return nil, err
	}
	if vc.IssuedAt > now || vc.ExpiresAt <= now || vc.ExpiresAt > vc.Permission.ValidUntil || vc.IssuedAt < vc.Permission.ValidFrom {
		return nil, errors.New("VC validity is outside the permission window")
	}
	if err := protocol.VerifyPermissionProofs(vc.Permission, vc.IssuerDID, anchor.PolicyHash, req.Proofs); err != nil {
		return nil, err
	}
	if err := verifyECDSAP256(issuer.SigningPublicKey, vc.Signature, mustCredentialBytes(vc)); err != nil {
		return nil, fmt.Errorf("credential signature: %w", err)
	}
	hash, err := protocol.CredentialHash(vc)
	if err != nil {
		return nil, err
	}
	proof := &protocol.VCProof{VCID: vc.ID, CredentialHash: hash, HolderDID: vc.HolderDID, IssuerDID: vc.IssuerDID, PolicyID: vc.PolicyID, Status: protocol.StatusValid, IssuedAt: vc.IssuedAt, ExpiresAt: vc.ExpiresAt, AnonymousEligible: vc.AnonymousEligible}
	if err := s.store.putJSON(prefixVC, vc.ID, proof); err != nil {
		return nil, err
	}
	sdk.Instance.EmitEvent(topicVCIssued, []string{vc.ID, vc.HolderDID, vc.PolicyID})
	return proof, nil
}

func (s *Service) getVCProof(vcID string) (*protocol.VCProof, error) {
	var proof protocol.VCProof
	if err := s.store.getJSON(prefixVC, vcID, &proof); err != nil {
		if errors.Is(err, errNotFound) {
			return nil, errors.New("VCProof not found")
		}
		return nil, err
	}
	return &proof, nil
}

func (s *Service) IssueNonce(req protocol.IssueNonceRequest) (*protocol.NonceRecord, error) {
	if _, err := s.requireSenderControls(req.VerifierDID); err != nil {
		return nil, err
	}
	if err := s.requireRole(req.VerifierDID, roleVerifier); err != nil {
		return nil, err
	}
	if err := validateID("nonce", req.Nonce); err != nil || req.Purpose == "" {
		return nil, errors.New("invalid nonce or purpose")
	}
	now, err := txTime()
	if err != nil {
		return nil, err
	}
	if req.ExpiresAt <= now {
		return nil, errors.New("nonce expiration must be in the future")
	}
	key := req.VerifierDID + "|" + req.Nonce
	if exists, err := s.store.exists(prefixNonce, key); err != nil || exists {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("nonce already exists")
	}
	record := &protocol.NonceRecord{VerifierDID: req.VerifierDID, Nonce: req.Nonce, Purpose: req.Purpose, ExpiresAt: req.ExpiresAt, Status: protocol.StatusActive}
	if err := s.store.putJSON(prefixNonce, key, record); err != nil {
		return nil, err
	}
	sdk.Instance.EmitEvent(topicNonceIssued, []string{req.VerifierDID, req.Nonce})
	return record, nil
}

func (s *Service) validateNonce(verifierDID, nonce, purpose string, now int64) (*protocol.NonceRecord, error) {
	var record protocol.NonceRecord
	key := verifierDID + "|" + nonce
	if err := s.store.getJSON(prefixNonce, key, &record); err != nil {
		return nil, errors.New("nonce not found")
	}
	if record.Status != protocol.StatusActive || record.ExpiresAt < now || record.Purpose != purpose {
		return nil, errors.New("nonce is expired, used, or purpose-mismatched")
	}
	return &record, nil
}

func (s *Service) consumeNonce(record *protocol.NonceRecord) error {
	record.Status = protocol.StatusUsed
	return s.store.putJSON(prefixNonce, record.VerifierDID+"|"+record.Nonce, record)
}

func (s *Service) VerifyVP(req protocol.VerifyVPRequest) error {
	if _, err := s.requireSenderControls(req.VP.VerifierDID); err != nil {
		return err
	}
	if err := s.requireRole(req.VP.VerifierDID, roleVerifier); err != nil {
		return err
	}
	if len(req.VP.Credentials) == 0 || len(req.VP.Credentials) != len(req.Presentations) {
		return errors.New("VP credential and presentation sets differ")
	}
	holder, err := s.requireActiveDID(req.VP.HolderDID)
	if err != nil {
		return err
	}
	vpBytes, err := protocol.VPBytes(req.VP)
	if err != nil {
		return err
	}
	if err := verifyECDSAP256(holder.SigningPublicKey, req.VP.Signature, vpBytes); err != nil {
		return fmt.Errorf("VP signature: %w", err)
	}
	now, err := txTime()
	if err != nil {
		return err
	}
	nonce, err := s.validateNonce(req.VP.VerifierDID, req.VP.Nonce, req.VP.Purpose, now)
	if err != nil {
		return err
	}
	permissionOK := req.Access == nil
	var currentAccess protocol.AccessRequest
	if req.Access != nil {
		currentAccess = *req.Access
		currentAccess.AtTime = now
	}
	for i, presentation := range req.Presentations {
		vc := presentation.Credential
		if vc.ID != req.VP.Credentials[i].ID || vc.HolderDID != req.VP.HolderDID {
			return errors.New("VP presentation order or holder binding mismatch")
		}
		signedHash, err := protocol.CredentialHash(req.VP.Credentials[i])
		if err != nil {
			return err
		}
		presentedHash, err := protocol.CredentialHash(vc)
		if err != nil || signedHash != presentedHash {
			return errors.New("VP signature does not bind the presented credential")
		}
		if _, err := s.validateCredential(vc, presentation.Proofs, now); err != nil {
			return fmt.Errorf("credential %s: %w", vc.ID, err)
		}
		if req.Access != nil && protocol.MatchPermission(currentAccess, vc.Permission) {
			permissionOK = true
		}
	}
	if !permissionOK {
		return errors.New("no credential permission matches access request")
	}
	if err := s.consumeNonce(nonce); err != nil {
		return err
	}
	sdk.Instance.EmitEvent(topicVPVerified, []string{req.VP.HolderDID, req.VP.VerifierDID, req.VP.Nonce})
	return nil
}
