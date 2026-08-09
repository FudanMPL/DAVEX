package service

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"didcontract/backend/internal/domain"
	"didcontract/protocol"
	"didcontract/sdk-go/revocation"
)

type CreateCommitteeInput struct {
	GroupID         string   `json:"groupID"`
	Threshold       int      `json:"threshold"`
	MemberDIDs      []string `json:"memberDIDs"`
	TermDurationSec int64    `json:"termDurationSeconds"`
}

type CommitteeMemberInput struct {
	GroupID         string `json:"groupID"`
	MemberDID       string `json:"memberDID"`
	TermDurationSec int64  `json:"termDurationSeconds"`
}

type CreateRevocationInput struct {
	DraftID        string `json:"draftID"`
	GroupID        string `json:"groupID"`
	VCID           string `json:"vcID"`
	EventIssuerDID string `json:"eventIssuerDID"`
	EventType      string `json:"eventType"`
	ScopeType      string `json:"scopeType"`
	ScopeID        string `json:"scopeID"`
}

type DraftInput struct {
	DraftID string `json:"draftID"`
}

func (s *Service) RegisterEventIssuer(ctx context.Context, actorAlias string, request protocol.RegisterEventIssuerRequest) (Result, error) {
	var entry protocol.EventIssuerEntry
	return s.call(ctx, actorAlias, "RegisterEventIssuer", request, &entry)
}

func (s *Service) RemoveEventIssuer(ctx context.Context, actorAlias string, request protocol.RegisterEventIssuerRequest) (Result, error) {
	var entry protocol.EventIssuerEntry
	return s.call(ctx, actorAlias, "RemoveEventIssuer", request, &entry)
}

func (s *Service) GetEventIssuer(ctx context.Context, actorAlias string, request protocol.RegisterEventIssuerRequest) (Result, error) {
	var entry protocol.EventIssuerEntry
	return s.call(ctx, actorAlias, "GetEventIssuer", request, &entry)
}

func (s *Service) CreateCommittee(ctx context.Context, actorAlias string, input CreateCommitteeInput) (Result, error) {
	var group protocol.CredentialGroup
	if _, err := s.call(ctx, actorAlias, "GetCredentialGroup", protocol.GroupRequest{GroupID: input.GroupID}, &group); err != nil {
		return Result{}, err
	}
	actor, err := s.actor(actorAlias)
	if err != nil {
		return Result{}, err
	}
	if err := requireActorDID(actor, group.IssuerDID); err != nil {
		return Result{}, err
	}
	if input.TermDurationSec <= 0 {
		input.TermDurationSec = 86400
	}
	now := s.now().Unix()
	members := make([]protocol.CommitteeMember, 0, len(input.MemberDIDs))
	seen := make(map[string]struct{}, len(input.MemberDIDs))
	for _, did := range input.MemberDIDs {
		did = strings.TrimSpace(did)
		if did == "" {
			return Result{}, badRequest("委员 DID 不能为空")
		}
		if _, exists := seen[did]; exists {
			return Result{}, badRequest("委员会包含重复 DID")
		}
		seen[did] = struct{}{}
		identity, _, err := s.store.Identity(did)
		if err != nil {
			return Result{}, notFound("本地委员身份不存在", err)
		}
		publicKey, err := identity.LSAGPublicKey()
		if err != nil {
			return Result{}, badRequest(err.Error())
		}
		members = append(members, protocol.CommitteeMember{DID: did, PublicKey: publicKey, TermStart: now - 5, TermEnd: now + input.TermDurationSec, Status: protocol.StatusActive})
	}
	request := protocol.CreateCommitteeRequest{GroupID: input.GroupID, Threshold: input.Threshold, Members: members}
	var committee protocol.RevocationCommittee
	return s.call(ctx, actorAlias, "CreateRevocationCommittee", request, &committee)
}

func (s *Service) AddCommitteeMember(ctx context.Context, actorAlias string, input CommitteeMemberInput) (Result, error) {
	if input.TermDurationSec <= 0 {
		input.TermDurationSec = 86400
	}
	identity, _, err := s.store.Identity(input.MemberDID)
	if err != nil {
		return Result{}, notFound("本地委员身份不存在", err)
	}
	publicKey, err := identity.LSAGPublicKey()
	if err != nil {
		return Result{}, badRequest(err.Error())
	}
	now := s.now().Unix()
	request := protocol.CommitteeMemberRequest{GroupID: input.GroupID, Member: protocol.CommitteeMember{DID: input.MemberDID, PublicKey: publicKey, TermStart: now - 5, TermEnd: now + input.TermDurationSec, Status: protocol.StatusActive}}
	var committee protocol.RevocationCommittee
	return s.call(ctx, actorAlias, "AddCommitteeMember", request, &committee)
}

func (s *Service) RemoveCommitteeMember(ctx context.Context, actorAlias string, input CommitteeMemberInput) (Result, error) {
	request := protocol.CommitteeMemberRequest{GroupID: input.GroupID, Member: protocol.CommitteeMember{DID: input.MemberDID}}
	var committee protocol.RevocationCommittee
	return s.call(ctx, actorAlias, "RemoveCommitteeMember", request, &committee)
}

func (s *Service) UpdateCommitteeThreshold(ctx context.Context, actorAlias string, request protocol.UpdateThresholdRequest) (Result, error) {
	var committee protocol.RevocationCommittee
	return s.call(ctx, actorAlias, "UpdateCommitteeThreshold", request, &committee)
}

func (s *Service) QueryCommittee(ctx context.Context, actorAlias, groupID string) (Result, error) {
	var committee protocol.RevocationCommittee
	return s.call(ctx, actorAlias, "GetRevocationCommittee", protocol.GroupRequest{GroupID: groupID}, &committee)
}

func (s *Service) QueryRotationLogs(ctx context.Context, actorAlias, groupID string) (Result, error) {
	var logs []protocol.RotationLog
	return s.call(ctx, actorAlias, "GetCommitteeRotationLogs", protocol.GroupRequest{GroupID: groupID}, &logs)
}

func (s *Service) CreateRevocationDraft(ctx context.Context, actorAlias string, input CreateRevocationInput) (Result, error) {
	actor, err := s.actor(actorAlias)
	if err != nil {
		return Result{}, err
	}
	if err := requireActorDID(actor, input.EventIssuerDID); err != nil {
		return Result{}, err
	}
	stored, err := s.store.Credential(input.VCID)
	if err != nil {
		return Result{}, notFound("本地目标 VC 不存在", err)
	}
	issuer, _, err := s.store.Identity(input.EventIssuerDID)
	if err != nil {
		return Result{}, notFound("本地事件签发方身份不存在", err)
	}
	holder, _, err := s.store.Identity(stored.Credential.HolderDID)
	if err != nil {
		return Result{}, notFound("本地 VC 持有者身份不存在", err)
	}
	holderPublicKey, err := holder.LSAGPublicKey()
	if err != nil {
		return Result{}, badRequest(err.Error())
	}
	if input.ScopeType == "" {
		input.ScopeType = protocol.ScopeGroup
	}
	if input.ScopeID == "" {
		input.ScopeID = input.GroupID
	}
	var proof protocol.VCProof
	if _, err := s.call(ctx, actorAlias, "GetVCProof", protocol.IDRequest{ID: input.VCID}, &proof); err != nil {
		return Result{}, err
	}
	if proof.Status != protocol.StatusValid || proof.HolderDID != stored.Credential.HolderDID {
		return Result{}, conflict("目标 VC 当前不是有效凭证", nil)
	}
	var group protocol.CredentialGroup
	if _, err := s.call(ctx, actorAlias, "GetCredentialGroup", protocol.GroupRequest{GroupID: input.GroupID}, &group); err != nil {
		return Result{}, err
	}
	if group.PolicyID != proof.PolicyID {
		return Result{}, conflict("目标 VC 与匿名群组策略不一致", nil)
	}
	var binding protocol.MemberBinding
	if _, err := s.call(ctx, actorAlias, "GetMemberBinding", protocol.MemberBindingRequest{GroupID: input.GroupID, VCID: input.VCID}, &binding); err != nil {
		return Result{}, err
	}
	if binding.Status != protocol.StatusActive || binding.HolderDID != stored.Credential.HolderDID || !strings.EqualFold(binding.PublicKey, holderPublicKey) {
		return Result{}, conflict("目标 VC 没有当前有效的群组成员绑定", nil)
	}
	if input.DraftID == "" {
		input.DraftID, err = randomID("revoke")
		if err != nil {
			return Result{}, err
		}
	}
	nonce, err := randomID("revnonce")
	if err != nil {
		return Result{}, err
	}
	now := s.now().Unix()
	credential := protocol.RevocationCredential{
		ID: input.DraftID, IssuerDID: input.EventIssuerDID, SubjectDID: stored.Credential.HolderDID,
		TargetVCID: input.VCID, EventType: input.EventType, EventTime: now, ScopeType: input.ScopeType, ScopeID: input.ScopeID,
	}
	credential, err = revocation.BuildCredential(issuer, credential)
	if err != nil {
		return Result{}, badRequest(err.Error())
	}
	draft := domain.RevocationDraft{
		ID: input.DraftID, Credential: credential, GroupID: input.GroupID, HolderDID: stored.Credential.HolderDID,
		HolderPublicKey: holderPublicKey, VCID: input.VCID, Nonce: nonce,
		Approvals: make(map[string]protocol.PartialSignature), Status: "Pending", CreatedAt: now, UpdatedAt: now,
	}
	if err := s.store.PutDraft(draft); err != nil {
		return Result{}, conflict("撤销草案已存在", err)
	}
	return Result{Data: draft}, nil
}

func (s *Service) GetRevocationDraft(id string) (Result, error) {
	draft, err := s.store.Draft(id)
	if err != nil {
		return Result{}, notFound("撤销草案不存在", err)
	}
	return Result{Data: draft}, nil
}

func (s *Service) ApproveRevocation(ctx context.Context, actorAlias, draftID string) (Result, error) {
	actor, err := s.actor(actorAlias)
	if err != nil {
		return Result{}, err
	}
	if actor.DID == "" {
		return Result{}, domain.NewError("ACTOR_DID_REQUIRED", http.StatusForbidden, "委员主体必须配置 DID")
	}
	draft, err := s.store.Draft(draftID)
	if err != nil {
		return Result{}, notFound("撤销草案不存在", err)
	}
	if draft.Status != "Pending" {
		return Result{}, conflict("撤销草案当前不可批准", nil)
	}
	var committee protocol.RevocationCommittee
	if _, err := s.call(ctx, actorAlias, "GetRevocationCommittee", protocol.GroupRequest{GroupID: draft.GroupID}, &committee); err != nil {
		return Result{}, err
	}
	identity, _, err := s.store.Identity(actor.DID)
	if err != nil {
		return Result{}, notFound("本地委员身份不存在", err)
	}
	publicKey, err := identity.LSAGPublicKey()
	if err != nil {
		return Result{}, badRequest(err.Error())
	}
	now := s.now().Unix()
	memberOK := false
	for _, member := range committee.Members {
		if member.DID == actor.DID && strings.EqualFold(member.PublicKey, publicKey) && member.Status == protocol.StatusActive && member.TermStart <= now && now < member.TermEnd {
			memberOK = true
		}
	}
	if !memberOK {
		return Result{}, domain.NewError("NOT_COMMITTEE_MEMBER", http.StatusForbidden, "当前主体不是有效委员会成员")
	}
	approval, err := revocation.Approve(identity, draft.Credential, draft.GroupID, draft.HolderDID, draft.VCID, draft.Nonce)
	if err != nil {
		return Result{}, badRequest(err.Error())
	}
	if err := revocation.VerifyApproval(publicKey, approval, draft.Credential, draft.GroupID, draft.HolderDID, draft.VCID, draft.Nonce); err != nil {
		return Result{}, domain.WrapError("APPROVAL_SELF_CHECK_FAILED", http.StatusInternalServerError, "批准签名本地自检失败", err)
	}
	updated, err := s.store.UpdateDraft(draftID, func(value *domain.RevocationDraft) error {
		if _, exists := value.Approvals[actor.DID]; exists {
			return conflict("同一委员不能重复批准", nil)
		}
		value.Approvals[actor.DID] = approval
		value.UpdatedAt = now
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return Result{Data: map[string]interface{}{"draft": updated, "approvalCount": len(updated.Approvals), "threshold": committee.Threshold}}, nil
}

func (s *Service) ExecuteRevocation(ctx context.Context, actorAlias, draftID string) (Result, error) {
	draft, err := s.store.Draft(draftID)
	if err != nil {
		return Result{}, notFound("撤销草案不存在", err)
	}
	if draft.Status != "Pending" {
		return Result{}, conflict("撤销草案已执行或不可执行", nil)
	}
	var committee protocol.RevocationCommittee
	if _, err := s.call(ctx, actorAlias, "GetRevocationCommittee", protocol.GroupRequest{GroupID: draft.GroupID}, &committee); err != nil {
		return Result{}, err
	}
	if len(draft.Approvals) < committee.Threshold {
		return Result{}, domain.NewError("THRESHOLD_NOT_MET", http.StatusConflict, "委员会批准数量尚未达到门限")
	}
	approverDIDs := make([]string, 0, len(draft.Approvals))
	for did := range draft.Approvals {
		approverDIDs = append(approverDIDs, did)
	}
	sort.Strings(approverDIDs)
	approvals := make([]protocol.PartialSignature, 0, len(approverDIDs))
	for _, did := range approverDIDs {
		approvals = append(approvals, draft.Approvals[did])
	}
	request := protocol.ExecuteRevocationRequest{
		Credential: draft.Credential, GroupID: draft.GroupID, HolderDID: draft.HolderDID,
		HolderPublicKey: draft.HolderPublicKey, VCID: draft.VCID, Nonce: draft.Nonce, Approvals: approvals,
	}
	var log protocol.RevocationLog
	result, err := s.call(ctx, actorAlias, "ExecuteRevocation", request, &log)
	if err != nil {
		return Result{}, err
	}
	txID := ""
	if result.Tx != nil {
		txID = result.Tx.TxID
	}
	_, updateErr := s.store.UpdateDraft(draftID, func(value *domain.RevocationDraft) error {
		value.Status = "Executed"
		value.UpdatedAt = s.now().Unix()
		value.ExecutedTxID = txID
		return nil
	})
	if updateErr != nil {
		return Result{}, conflict("链上撤销成功但本地草案状态保存失败", updateErr)
	}
	var proof protocol.VCProof
	_, _ = s.call(ctx, actorAlias, "GetVCProof", protocol.IDRequest{ID: draft.VCID}, &proof)
	var group protocol.CredentialGroup
	_, _ = s.call(ctx, actorAlias, "GetCredentialGroup", protocol.GroupRequest{GroupID: draft.GroupID}, &group)
	result.Data = map[string]interface{}{"log": log, "vcProof": proof, "group": group, "credentialHash": protocol.RevocationCredentialHash(draft.Credential)}
	return result, nil
}

func (s *Service) QueryRevocationLogs(ctx context.Context, actorAlias, vcID string) (Result, error) {
	var logs []protocol.RevocationLog
	result, err := s.call(ctx, actorAlias, "GetRevocationLogs", protocol.IDRequest{ID: vcID}, &logs)
	if err != nil {
		return Result{}, err
	}
	var proof protocol.VCProof
	_, _ = s.call(ctx, actorAlias, "GetVCProof", protocol.IDRequest{ID: vcID}, &proof)
	result.Data = map[string]interface{}{"logs": logs, "vcProof": proof}
	return result, nil
}

func (s *Service) IsCredentialConsumed(ctx context.Context, actorAlias, hash string) (Result, error) {
	var state struct {
		Consumed bool `json:"consumed"`
	}
	return s.call(ctx, actorAlias, "IsCredentialConsumed", protocol.IDRequest{ID: hash}, &state)
}
