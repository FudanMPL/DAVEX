package main

import (
	"errors"
	"fmt"
	"sort"

	"chainmaker.org/chainmaker/contract-sdk-go/v2/sdk"
	"didcontract/protocol"
)

func eventIssuerKey(eventType, issuerDID string) string {
	return eventType + "|" + issuerDID
}

func (s *Service) RegisterEventIssuer(req protocol.RegisterEventIssuerRequest) (*protocol.EventIssuerEntry, error) {
	if _, err := s.requireGovernanceSender(); err != nil {
		return nil, err
	}
	if err := validateID("event type", req.EventType); err != nil || req.MaxAgeSeconds <= 0 {
		return nil, errors.New("invalid event type or maximum age")
	}
	if _, err := s.requireActiveDID(req.IssuerDID); err != nil {
		return nil, err
	}
	now, err := txTime()
	if err != nil {
		return nil, err
	}
	entry := &protocol.EventIssuerEntry{EventType: req.EventType, IssuerDID: req.IssuerDID, MaxAgeSeconds: req.MaxAgeSeconds, Status: protocol.StatusActive, UpdatedAt: now}
	if err := s.store.putJSON(prefixEventIssuer, eventIssuerKey(req.EventType, req.IssuerDID), entry); err != nil {
		return nil, err
	}
	sdk.Instance.EmitEvent(topicEventIssuerUpdated, []string{req.EventType, req.IssuerDID, protocol.StatusActive})
	return entry, nil
}

func (s *Service) RemoveEventIssuer(req protocol.RegisterEventIssuerRequest) (*protocol.EventIssuerEntry, error) {
	if _, err := s.requireGovernanceSender(); err != nil {
		return nil, err
	}
	entry, err := s.getEventIssuer(req.EventType, req.IssuerDID)
	if err != nil {
		return nil, err
	}
	now, err := txTime()
	if err != nil {
		return nil, err
	}
	entry.Status = protocol.StatusInactive
	entry.UpdatedAt = now
	if err := s.store.putJSON(prefixEventIssuer, eventIssuerKey(req.EventType, req.IssuerDID), entry); err != nil {
		return nil, err
	}
	sdk.Instance.EmitEvent(topicEventIssuerUpdated, []string{req.EventType, req.IssuerDID, protocol.StatusInactive})
	return entry, nil
}

func (s *Service) getEventIssuer(eventType, issuerDID string) (*protocol.EventIssuerEntry, error) {
	var entry protocol.EventIssuerEntry
	if err := s.store.getJSON(prefixEventIssuer, eventIssuerKey(eventType, issuerDID), &entry); err != nil {
		if errors.Is(err, errNotFound) {
			return nil, errors.New("event issuer is not registered")
		}
		return nil, err
	}
	return &entry, nil
}

func validateThreshold(threshold, activeMembers int) error {
	if activeMembers < 2 || threshold < (activeMembers+1)/2 || threshold > activeMembers {
		return errors.New("threshold must be a majority and not exceed active member count")
	}
	return nil
}

func (s *Service) validateCommitteeMember(member protocol.CommitteeMember, now int64) (protocol.CommitteeMember, error) {
	if err := validateID("committee member DID", member.DID); err != nil {
		return protocol.CommitteeMember{}, err
	}
	did, err := s.requireActiveDID(member.DID)
	if err != nil {
		return protocol.CommitteeMember{}, err
	}
	if normalizeHex(member.PublicKey) != normalizeHex(did.LSAGPublicKey) {
		return protocol.CommitteeMember{}, errors.New("committee public key is not bound to member DID")
	}
	if member.TermStart > now || member.TermEnd <= now {
		return protocol.CommitteeMember{}, errors.New("committee member term is not currently valid")
	}
	member.PublicKey = normalizeHex(member.PublicKey)
	member.Status = protocol.StatusActive
	return member, nil
}

func (s *Service) getCommittee(groupID string) (*protocol.RevocationCommittee, error) {
	var committee protocol.RevocationCommittee
	if err := s.store.getJSON(prefixCommittee, groupID, &committee); err != nil {
		if errors.Is(err, errNotFound) {
			return nil, errors.New("revocation committee not found")
		}
		return nil, err
	}
	return &committee, nil
}

func (s *Service) CreateRevocationCommittee(req protocol.CreateCommitteeRequest) (*protocol.RevocationCommittee, error) {
	group, err := s.getGroup(req.GroupID)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireSenderControls(group.IssuerDID); err != nil {
		return nil, err
	}
	if exists, err := s.store.exists(prefixCommittee, req.GroupID); err != nil || exists {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("revocation committee already exists")
	}
	if len(req.Members) > maxCommitteeSize {
		return nil, errors.New("committee exceeds maximum size")
	}
	if err := validateThreshold(req.Threshold, len(req.Members)); err != nil {
		return nil, err
	}
	now, err := txTime()
	if err != nil {
		return nil, err
	}
	members := make([]protocol.CommitteeMember, 0, len(req.Members))
	seen := make(map[string]struct{}, len(req.Members))
	for _, candidate := range req.Members {
		if _, ok := seen[candidate.DID]; ok {
			return nil, errors.New("duplicate committee member DID")
		}
		seen[candidate.DID] = struct{}{}
		member, err := s.validateCommitteeMember(candidate, now)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	sort.Slice(members, func(i, j int) bool { return members[i].DID < members[j].DID })
	committee := &protocol.RevocationCommittee{GroupID: req.GroupID, IssuerDID: group.IssuerDID, Threshold: req.Threshold, Members: members, Epoch: 0, Status: protocol.StatusActive, CreatedAt: now, UpdatedAt: now}
	if err := s.store.putJSON(prefixCommittee, req.GroupID, committee); err != nil {
		return nil, err
	}
	if err := s.writeRotationLog(committee, "create", "", group.IssuerDID, now); err != nil {
		return nil, err
	}
	sdk.Instance.EmitEvent(topicCommitteeUpdated, []string{req.GroupID, "create"})
	return committee, nil
}

func activeMemberCount(members []protocol.CommitteeMember) int {
	count := 0
	for _, member := range members {
		if member.Status == protocol.StatusActive {
			count++
		}
	}
	return count
}

func (s *Service) AddCommitteeMember(req protocol.CommitteeMemberRequest) (*protocol.RevocationCommittee, error) {
	committee, err := s.getCommittee(req.GroupID)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireSenderControls(committee.IssuerDID); err != nil {
		return nil, err
	}
	now, err := txTime()
	if err != nil {
		return nil, err
	}
	member, err := s.validateCommitteeMember(req.Member, now)
	if err != nil {
		return nil, err
	}
	found := false
	for i := range committee.Members {
		if committee.Members[i].DID == member.DID {
			committee.Members[i] = member
			found = true
			break
		}
	}
	if !found {
		if len(committee.Members) >= maxCommitteeSize {
			return nil, errors.New("committee reached maximum size")
		}
		committee.Members = append(committee.Members, member)
	}
	sort.Slice(committee.Members, func(i, j int) bool { return committee.Members[i].DID < committee.Members[j].DID })
	committee.Epoch++
	committee.UpdatedAt = now
	if err := s.store.putJSON(prefixCommittee, req.GroupID, committee); err != nil {
		return nil, err
	}
	if err := s.writeRotationLog(committee, "upsert_member", member.DID, committee.IssuerDID, now); err != nil {
		return nil, err
	}
	sdk.Instance.EmitEvent(topicCommitteeUpdated, []string{req.GroupID, "upsert_member", member.DID})
	return committee, nil
}

func (s *Service) RemoveCommitteeMember(req protocol.CommitteeMemberRequest) (*protocol.RevocationCommittee, error) {
	committee, err := s.getCommittee(req.GroupID)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireSenderControls(committee.IssuerDID); err != nil {
		return nil, err
	}
	found := false
	for i := range committee.Members {
		if committee.Members[i].DID == req.Member.DID && committee.Members[i].Status == protocol.StatusActive {
			committee.Members[i].Status = protocol.StatusInactive
			found = true
			break
		}
	}
	if !found {
		return nil, errors.New("active committee member not found")
	}
	if err := validateThreshold(committee.Threshold, activeMemberCount(committee.Members)); err != nil {
		return nil, err
	}
	now, err := txTime()
	if err != nil {
		return nil, err
	}
	committee.Epoch++
	committee.UpdatedAt = now
	if err := s.store.putJSON(prefixCommittee, req.GroupID, committee); err != nil {
		return nil, err
	}
	if err := s.writeRotationLog(committee, "remove_member", req.Member.DID, committee.IssuerDID, now); err != nil {
		return nil, err
	}
	sdk.Instance.EmitEvent(topicCommitteeUpdated, []string{req.GroupID, "remove_member", req.Member.DID})
	return committee, nil
}

func (s *Service) UpdateCommitteeThreshold(req protocol.UpdateThresholdRequest) (*protocol.RevocationCommittee, error) {
	committee, err := s.getCommittee(req.GroupID)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireSenderControls(committee.IssuerDID); err != nil {
		return nil, err
	}
	if err := validateThreshold(req.Threshold, activeMemberCount(committee.Members)); err != nil {
		return nil, err
	}
	now, err := txTime()
	if err != nil {
		return nil, err
	}
	committee.Threshold = req.Threshold
	committee.Epoch++
	committee.UpdatedAt = now
	if err := s.store.putJSON(prefixCommittee, req.GroupID, committee); err != nil {
		return nil, err
	}
	if err := s.writeRotationLog(committee, "update_threshold", "", committee.IssuerDID, now); err != nil {
		return nil, err
	}
	sdk.Instance.EmitEvent(topicCommitteeUpdated, []string{req.GroupID, "update_threshold"})
	return committee, nil
}

func (s *Service) writeRotationLog(committee *protocol.RevocationCommittee, operation, memberDID, operator string, now int64) error {
	id := txID() + "|" + operation + "|" + memberDID
	log := &protocol.RotationLog{ID: id, GroupID: committee.GroupID, Operation: operation, MemberDID: memberDID, Threshold: committee.Threshold, Epoch: committee.Epoch, Operator: operator, CreatedAt: now}
	if err := s.store.putJSON(prefixRotationLog, id, log); err != nil {
		return err
	}
	return s.store.appendIndex(prefixRotationIdx, committee.GroupID, id)
}

func (s *Service) validateRevocationCredential(req protocol.ExecuteRevocationRequest, group *protocol.CredentialGroup, binding *protocol.MemberBinding, now int64) (string, error) {
	credential := req.Credential
	if err := validateID("revocation credential ID", credential.ID); err != nil {
		return "", err
	}
	if credential.SubjectDID != req.HolderDID || credential.TargetVCID != req.VCID || credential.SubjectDID != binding.HolderDID {
		return "", errors.New("revocation credential target binding mismatch")
	}
	if credential.ScopeType == protocol.ScopeGroup {
		if credential.ScopeID != req.GroupID {
			return "", errors.New("group-scoped revocation credential mismatch")
		}
	} else if credential.ScopeType == protocol.ScopePolicy {
		if credential.ScopeID != group.PolicyID {
			return "", errors.New("policy-scoped revocation credential mismatch")
		}
	} else {
		return "", errors.New("unsupported revocation scope type")
	}
	entry, err := s.getEventIssuer(credential.EventType, credential.IssuerDID)
	if err != nil || entry.Status != protocol.StatusActive {
		return "", errors.New("revocation event issuer is not active")
	}
	if credential.EventTime > now || now-credential.EventTime > entry.MaxAgeSeconds {
		return "", errors.New("revocation credential is outside event validity window")
	}
	issuer, err := s.requireActiveDID(credential.IssuerDID)
	if err != nil {
		return "", err
	}
	if err := verifyECDSAP256(issuer.SigningPublicKey, credential.Signature, protocol.RevocationCredentialBytes(credential)); err != nil {
		return "", fmt.Errorf("revocation credential signature: %w", err)
	}
	credentialHash := protocol.RevocationCredentialHash(credential)
	if exists, err := s.store.exists(prefixConsumed, credentialHash); err != nil || exists {
		if err != nil {
			return "", err
		}
		return "", errors.New("revocation credential has already been consumed")
	}
	return credentialHash, nil
}

func (s *Service) ExecuteRevocation(req protocol.ExecuteRevocationRequest) (*protocol.RevocationLog, error) {
	if err := validateID("revocation nonce", req.Nonce); err != nil {
		return nil, err
	}
	group, binding, err := s.validateGroupBinding(req.GroupID, req.VCID, req.HolderDID, req.HolderPublicKey)
	if err != nil {
		return nil, err
	}
	vc, err := s.getVCProof(req.VCID)
	if err != nil || vc.Status != protocol.StatusValid || vc.HolderDID != req.HolderDID || vc.PolicyID != group.PolicyID {
		return nil, errors.New("target VCProof is not a valid member credential")
	}
	now, err := txTime()
	if err != nil {
		return nil, err
	}
	credentialHash, err := s.validateRevocationCredential(req, group, binding, now)
	if err != nil {
		return nil, err
	}
	committee, err := s.getCommittee(req.GroupID)
	if err != nil || committee.Status != protocol.StatusActive || committee.IssuerDID != group.IssuerDID {
		return nil, errors.New("revocation committee is not active or group-bound")
	}
	if len(req.Approvals) < committee.Threshold || len(req.Approvals) > len(committee.Members) {
		return nil, errors.New("approval set cannot meet committee threshold")
	}
	memberByDID := make(map[string]protocol.CommitteeMember, len(committee.Members))
	for _, member := range committee.Members {
		memberByDID[member.DID] = member
	}
	message := protocol.RevocationMessage(credentialHash, req.GroupID, req.HolderDID, req.VCID, req.Nonce)
	seen := make(map[string]struct{}, len(req.Approvals))
	approvers := make([]string, 0, len(req.Approvals))
	for _, approval := range req.Approvals {
		if approval.MemberDID == req.HolderDID {
			return nil, errors.New("revocation subject cannot approve own revocation")
		}
		if _, duplicate := seen[approval.MemberDID]; duplicate {
			return nil, errors.New("duplicate committee approver")
		}
		member, ok := memberByDID[approval.MemberDID]
		if !ok || member.Status != protocol.StatusActive || member.TermStart > now || member.TermEnd <= now {
			return nil, errors.New("approver is not a current committee member")
		}
		registry, err := s.requireActiveDID(member.DID)
		if err != nil || normalizeHex(registry.LSAGPublicKey) != normalizeHex(member.PublicKey) {
			return nil, errors.New("approver DID is inactive or its current committee key does not match")
		}
		if err := verifySchnorrRistretto(member.PublicKey, approval.R, approval.S, message); err != nil {
			return nil, fmt.Errorf("approval by %s: %w", approval.MemberDID, err)
		}
		seen[approval.MemberDID] = struct{}{}
		approvers = append(approvers, approval.MemberDID)
	}
	if len(approvers) < committee.Threshold {
		return nil, errors.New("valid unique approvals do not meet threshold")
	}
	sort.Strings(approvers)
	if err := s.revokeMemberBinding(group, binding, now); err != nil {
		return nil, err
	}
	vc.Status = protocol.StatusRevoked
	if err := s.store.putJSON(prefixVC, vc.VCID, vc); err != nil {
		return nil, err
	}
	if err := s.store.putString(prefixConsumed, credentialHash, req.Credential.ID); err != nil {
		return nil, err
	}
	logID := txID()
	log := &protocol.RevocationLog{ID: logID, CredentialHash: credentialHash, CredentialID: req.Credential.ID, GroupID: req.GroupID, VCID: req.VCID, HolderDID: req.HolderDID, EventType: req.Credential.EventType, EventTime: req.Credential.EventTime, Approvers: approvers, ExecutedAt: now, TxID: logID}
	if err := s.store.putJSON(prefixRevocationLog, logID, log); err != nil {
		return nil, err
	}
	if err := s.store.appendIndex(prefixRevocationIdx, req.VCID, logID); err != nil {
		return nil, err
	}
	sdk.Instance.EmitEvent(topicCredentialRevoked, []string{req.VCID, req.GroupID, req.HolderDID, credentialHash})
	return log, nil
}

func (s *Service) IsCredentialConsumed(credentialHash string) (bool, error) {
	if _, err := protocol.DecodeHash(credentialHash); err != nil {
		return false, err
	}
	return s.store.exists(prefixConsumed, normalizeHex(credentialHash))
}

func (s *Service) GetRevocationLogs(vcID string) ([]protocol.RevocationLog, error) {
	ids, err := s.store.getIndex(prefixRevocationIdx, vcID)
	if err != nil {
		return nil, err
	}
	logs := make([]protocol.RevocationLog, 0, len(ids))
	for _, id := range ids {
		var log protocol.RevocationLog
		if err := s.store.getJSON(prefixRevocationLog, id, &log); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, nil
}

func (s *Service) GetRotationLogs(groupID string) ([]protocol.RotationLog, error) {
	ids, err := s.store.getIndex(prefixRotationIdx, groupID)
	if err != nil {
		return nil, err
	}
	logs := make([]protocol.RotationLog, 0, len(ids))
	for _, id := range ids {
		var log protocol.RotationLog
		if err := s.store.getJSON(prefixRotationLog, id, &log); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, nil
}
