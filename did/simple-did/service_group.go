package main

import (
	"errors"
	"fmt"
	"sort"

	"chainmaker.org/chainmaker/contract-sdk-go/v2/sdk"
	"didcontract/protocol"
)

func (s *Service) getGroup(groupID string) (*protocol.CredentialGroup, error) {
	var group protocol.CredentialGroup
	if err := s.store.getJSON(prefixGroup, groupID, &group); err != nil {
		if errors.Is(err, errNotFound) {
			return nil, errors.New("credential group not found")
		}
		return nil, err
	}
	return &group, nil
}

func (s *Service) CreateCredentialGroup(req protocol.CreateGroupRequest) (*protocol.CredentialGroup, error) {
	if err := validateID("group ID", req.GroupID); err != nil {
		return nil, err
	}
	if _, err := s.requireSenderControls(req.IssuerDID); err != nil {
		return nil, err
	}
	if err := s.requireRole(req.IssuerDID, roleCredentialIssuer); err != nil {
		return nil, err
	}
	if exists, err := s.store.exists(prefixGroup, req.GroupID); err != nil || exists {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("credential group already exists")
	}
	anchor, err := s.getPolicy(req.PolicyID)
	if err != nil || anchor.Status != protocol.StatusActive || anchor.IssuerDID != req.IssuerDID {
		return nil, errors.New("group must bind an active policy owned by issuer")
	}
	if req.Qualification.PolicyID != req.PolicyID {
		return nil, errors.New("qualification policy ID mismatch")
	}
	if err := protocol.VerifyPermissionProofs(req.Qualification, req.IssuerDID, anchor.PolicyHash, req.Proofs); err != nil {
		return nil, err
	}
	qID, err := protocol.QualificationID(req.Qualification, req.IssuerDID)
	if err != nil {
		return nil, err
	}
	now, err := txTime()
	if err != nil {
		return nil, err
	}
	group := &protocol.CredentialGroup{
		GroupID: req.GroupID, PolicyID: req.PolicyID, QID: qID, IssuerDID: req.IssuerDID,
		PublicKeys: []string{}, MemberEpoch: 0, LHash: protocol.GroupHash(nil, 0),
		Status: protocol.StatusActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.store.putJSON(prefixGroup, req.GroupID, group); err != nil {
		return nil, err
	}
	sdk.Instance.EmitEvent(topicGroupCreated, []string{req.GroupID, req.PolicyID, qID})
	return group, nil
}

func (s *Service) AddMemberToGroup(req protocol.AddGroupMemberRequest) (*protocol.MemberBinding, error) {
	group, err := s.getGroup(req.GroupID)
	if err != nil {
		return nil, err
	}
	if group.Status != protocol.StatusActive {
		return nil, errors.New("credential group is not active")
	}
	if _, err := s.requireSenderControls(group.IssuerDID); err != nil {
		return nil, err
	}
	if err := s.requireRole(group.IssuerDID, roleCredentialIssuer); err != nil {
		return nil, err
	}
	now, err := txTime()
	if err != nil {
		return nil, err
	}
	stored, err := s.validateCredential(req.Credential, req.Proofs, now)
	if err != nil {
		return nil, err
	}
	if !stored.AnonymousEligible || req.Credential.PolicyID != group.PolicyID || req.Credential.IssuerDID != group.IssuerDID {
		return nil, errors.New("credential is not eligible for this anonymous group")
	}
	qID, err := protocol.QualificationID(req.Credential.Permission, req.Credential.IssuerDID)
	if err != nil || qID != group.QID {
		return nil, errors.New("credential qualification does not match group qID")
	}
	holder, err := s.requireActiveDID(req.Credential.HolderDID)
	if err != nil {
		return nil, err
	}
	publicKey := normalizeHex(req.PublicKey)
	if publicKey != normalizeHex(holder.LSAGPublicKey) {
		return nil, errors.New("group public key is not bound to credential holder DID")
	}
	if len(group.PublicKeys) >= maxRingSize {
		return nil, errors.New("credential group reached maximum ring size")
	}
	bindingKey := req.GroupID + "|" + req.Credential.ID
	if exists, err := s.store.exists(prefixBinding, bindingKey); err != nil || exists {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("VC already has a group membership binding")
	}
	pubKeyIndex := req.GroupID + "|" + publicKey
	if exists, err := s.store.exists(prefixBindingPub, pubKeyIndex); err != nil || exists {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("public key already belongs to the group")
	}
	binding := &protocol.MemberBinding{GroupID: req.GroupID, VCID: req.Credential.ID, HolderDID: req.Credential.HolderDID, PublicKey: publicKey, Status: protocol.StatusActive, JoinedAt: now, UpdatedAt: now}
	group.PublicKeys = append(group.PublicKeys, publicKey)
	sort.Strings(group.PublicKeys)
	group.MemberEpoch++
	group.LHash = protocol.GroupHash(group.PublicKeys, group.MemberEpoch)
	group.UpdatedAt = now
	if err := s.store.putJSON(prefixBinding, bindingKey, binding); err != nil {
		return nil, err
	}
	if err := s.store.putJSON(prefixBindingPub, pubKeyIndex, binding); err != nil {
		return nil, err
	}
	if err := s.store.putJSON(prefixGroup, group.GroupID, group); err != nil {
		return nil, err
	}
	sdk.Instance.EmitEvent(topicGroupMemberAdded, []string{req.GroupID, req.Credential.ID, req.Credential.HolderDID})
	return binding, nil
}

func (s *Service) getMemberBinding(groupID, vcID string) (*protocol.MemberBinding, error) {
	var binding protocol.MemberBinding
	if err := s.store.getJSON(prefixBinding, groupID+"|"+vcID, &binding); err != nil {
		if errors.Is(err, errNotFound) {
			return nil, errors.New("member binding not found")
		}
		return nil, err
	}
	return &binding, nil
}

func (s *Service) SetGroupStatus(groupID, status string) (*protocol.CredentialGroup, error) {
	group, err := s.getGroup(groupID)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireSenderControls(group.IssuerDID); err != nil {
		return nil, err
	}
	if status != protocol.StatusActive && status != protocol.StatusInactive {
		return nil, errors.New("unsupported group status")
	}
	if status == protocol.StatusActive {
		anchor, err := s.getPolicy(group.PolicyID)
		if err != nil || anchor.Status != protocol.StatusActive {
			return nil, errors.New("cannot activate a group whose policy is inactive")
		}
	}
	now, err := txTime()
	if err != nil {
		return nil, err
	}
	group.Status = status
	group.UpdatedAt = now
	if err := s.store.putJSON(prefixGroup, groupID, group); err != nil {
		return nil, err
	}
	sdk.Instance.EmitEvent(topicGroupStatusChanged, []string{groupID, status})
	return group, nil
}

func (s *Service) CheckKeyImage(keyImage string) (*protocol.KeyImageRecord, bool, error) {
	keyImage = normalizeHex(keyImage)
	if err := validateLSAGPoint(keyImage); err != nil {
		return nil, false, err
	}
	var record protocol.KeyImageRecord
	if err := s.store.getJSON(prefixKeyImage, keyImage, &record); err != nil {
		if errors.Is(err, errNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &record, true, nil
}

func (s *Service) VerifyPrivacyVP(req protocol.VerifyPrivacyVPRequest) error {
	vp := req.VP
	group, err := s.getGroup(vp.GroupID)
	if err != nil {
		return err
	}
	if group.Status != protocol.StatusActive || len(group.PublicKeys) == 0 {
		return errors.New("credential group is inactive or empty")
	}
	anchor, err := s.getPolicy(group.PolicyID)
	if err != nil || anchor.Status != protocol.StatusActive {
		return errors.New("group policy is not active")
	}
	qID, err := protocol.QualificationID(req.Qualification, group.IssuerDID)
	if err != nil || qID != group.QID {
		return errors.New("requested qualification does not match group qID")
	}
	ctx := vp.Context
	if vp.PolicyID != group.PolicyID || vp.QID != group.QID || vp.LHash != group.LHash ||
		ctx.GroupID != group.GroupID || ctx.PolicyID != group.PolicyID || ctx.QID != group.QID || ctx.LHash != group.LHash {
		return errors.New("privacy VP context does not match current group anchor")
	}
	if _, err := s.requireSenderControls(ctx.VerifierDID); err != nil {
		return err
	}
	if err := s.requireRole(ctx.VerifierDID, roleVerifier); err != nil {
		return err
	}
	now, err := txTime()
	if err != nil {
		return err
	}
	currentAccess := req.Access
	currentAccess.AtTime = now
	if !protocol.MatchPermission(currentAccess, req.Qualification) {
		return errors.New("requested access is not currently authorized by the qualification")
	}
	if ctx.TimeSlot != now/timeSlotSeconds {
		return errors.New("privacy VP time slot is not current")
	}
	nonce, err := s.validateNonce(ctx.VerifierDID, ctx.Nonce, ctx.Purpose, now)
	if err != nil {
		return err
	}
	keyImage := normalizeHex(vp.Signature.KeyImage)
	if _, used, err := s.CheckKeyImage(keyImage); err != nil {
		return err
	} else if used {
		return errors.New("contextual key image has already been used")
	}
	if err := verifyLSAG(group.PublicKeys, ctx, vp.Signature); err != nil {
		return err
	}
	if err := s.consumeNonce(nonce); err != nil {
		return err
	}
	record := &protocol.KeyImageRecord{KeyImage: keyImage, GroupID: group.GroupID, PolicyID: group.PolicyID, QID: group.QID, TimeSlot: ctx.TimeSlot, UsedAt: now, TxID: txID()}
	if err := s.store.putJSON(prefixKeyImage, keyImage, record); err != nil {
		return err
	}
	sdk.Instance.EmitEvent(topicPrivacyVPVerified, []string{vp.ID, group.GroupID, keyImage})
	return nil
}

func removePublicKey(values []string, target string) ([]string, bool) {
	result := make([]string, 0, len(values))
	removed := false
	for _, value := range values {
		if normalizeHex(value) == normalizeHex(target) {
			removed = true
			continue
		}
		result = append(result, value)
	}
	return result, removed
}

func (s *Service) revokeMemberBinding(group *protocol.CredentialGroup, binding *protocol.MemberBinding, now int64) error {
	updatedKeys, removed := removePublicKey(group.PublicKeys, binding.PublicKey)
	if !removed {
		return errors.New("bound public key is absent from group")
	}
	binding.Status = protocol.StatusRevoked
	binding.UpdatedAt = now
	group.PublicKeys = updatedKeys
	group.MemberEpoch++
	group.LHash = protocol.GroupHash(group.PublicKeys, group.MemberEpoch)
	group.UpdatedAt = now
	if err := s.store.putJSON(prefixBinding, group.GroupID+"|"+binding.VCID, binding); err != nil {
		return err
	}
	if err := s.store.putJSON(prefixBindingPub, group.GroupID+"|"+binding.PublicKey, binding); err != nil {
		return err
	}
	return s.store.putJSON(prefixGroup, group.GroupID, group)
}

func (s *Service) validateGroupBinding(groupID, vcID, holderDID, publicKey string) (*protocol.CredentialGroup, *protocol.MemberBinding, error) {
	group, err := s.getGroup(groupID)
	if err != nil {
		return nil, nil, err
	}
	binding, err := s.getMemberBinding(groupID, vcID)
	if err != nil {
		return nil, nil, err
	}
	if binding.Status != protocol.StatusActive || binding.HolderDID != holderDID || normalizeHex(binding.PublicKey) != normalizeHex(publicKey) {
		return nil, nil, fmt.Errorf("member binding tuple does not match (%s,%s,%s,PK)", groupID, vcID, holderDID)
	}
	return group, binding, nil
}
