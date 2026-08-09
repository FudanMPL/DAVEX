package service

import (
	"context"
	"net/http"
	"strings"

	"didcontract/backend/internal/domain"
	"didcontract/protocol"
	"didcontract/sdk-go/govdid"
	"didcontract/sdk-go/lsag"
)

type CreateGroupInput struct {
	GroupID    string `json:"groupID"`
	PolicyID   string `json:"policyID"`
	IssuerDID  string `json:"issuerDID"`
	EntryIndex int    `json:"entryIndex"`
}

type AddGroupMemberInput struct {
	GroupID string `json:"groupID"`
	VCID    string `json:"vcID"`
}

type GroupStatusInput struct {
	GroupID string `json:"groupID"`
	Status  string `json:"status"`
}

type GeneratePrivacyVPInput struct {
	VPID        string `json:"vpID"`
	GroupID     string `json:"groupID"`
	HolderDID   string `json:"holderDID"`
	VerifierDID string `json:"verifierDID"`
	Nonce       string `json:"nonce"`
	Purpose     string `json:"purpose"`
}

type KeyImageInput struct {
	GroupID   string `json:"groupID"`
	HolderDID string `json:"holderDID"`
}

func (s *Service) CreateGroup(ctx context.Context, actorAlias string, input CreateGroupInput) (Result, error) {
	actor, err := s.actor(actorAlias)
	if err != nil {
		return Result{}, err
	}
	if err := requireActorDID(actor, input.IssuerDID); err != nil {
		return Result{}, err
	}
	bundle, err := s.store.Policy(input.PolicyID)
	if err != nil {
		return Result{}, notFound("本地策略不存在", err)
	}
	if bundle.IssuerDID != input.IssuerDID {
		return Result{}, badRequest("群组签发方与策略签发方不一致")
	}
	entry, err := bundle.Entry(input.EntryIndex)
	if err != nil {
		return Result{}, badRequest(err.Error())
	}
	request := protocol.CreateGroupRequest{GroupID: input.GroupID, PolicyID: input.PolicyID, IssuerDID: input.IssuerDID, Qualification: entry.Permission, Proofs: entry.Proofs}
	var group protocol.CredentialGroup
	result, err := s.call(ctx, actorAlias, "CreateCredentialGroup", request, &group)
	if err != nil {
		return Result{}, err
	}
	if err := s.store.PutGroup(input.GroupID); err != nil {
		return Result{}, conflict("群组已上链但本地索引保存失败", err)
	}
	return result, nil
}

func (s *Service) AddGroupMember(ctx context.Context, actorAlias string, input AddGroupMemberInput) (Result, error) {
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
	stored, err := s.store.Credential(input.VCID)
	if err != nil {
		return Result{}, notFound("本地完整 VC 不存在", err)
	}
	holder, _, err := s.store.Identity(stored.Credential.HolderDID)
	if err != nil {
		return Result{}, notFound("本地持有者身份不存在", err)
	}
	publicKey, err := holder.LSAGPublicKey()
	if err != nil {
		return Result{}, badRequest(err.Error())
	}
	request := protocol.AddGroupMemberRequest{GroupID: input.GroupID, Credential: stored.Credential, Proofs: stored.Proofs, PublicKey: publicKey}
	var binding protocol.MemberBinding
	return s.call(ctx, actorAlias, "AddMemberToGroup", request, &binding)
}

func (s *Service) QueryGroup(ctx context.Context, actorAlias, groupID string) (Result, error) {
	var group protocol.CredentialGroup
	return s.call(ctx, actorAlias, "GetCredentialGroup", protocol.GroupRequest{GroupID: groupID}, &group)
}

func (s *Service) SetGroupStatus(ctx context.Context, actorAlias string, input GroupStatusInput) (Result, error) {
	method := "SuspendGroup"
	if strings.EqualFold(input.Status, protocol.StatusActive) {
		method = "ActivateGroup"
	}
	var group protocol.CredentialGroup
	return s.call(ctx, actorAlias, method, protocol.GroupRequest{GroupID: input.GroupID}, &group)
}

func (s *Service) GeneratePrivacyVP(ctx context.Context, actorAlias string, input GeneratePrivacyVPInput) (Result, error) {
	actor, err := s.actor(actorAlias)
	if err != nil {
		return Result{}, err
	}
	if err := requireActorDID(actor, input.HolderDID); err != nil {
		return Result{}, err
	}
	var group protocol.CredentialGroup
	if _, err := s.call(ctx, actorAlias, "GetCredentialGroup", protocol.GroupRequest{GroupID: input.GroupID}, &group); err != nil {
		return Result{}, err
	}
	if group.Status != protocol.StatusActive {
		return Result{}, domain.NewError("GROUP_INACTIVE", http.StatusConflict, "匿名认证群组未激活")
	}
	entry, err := s.qualificationEntry(group)
	if err != nil {
		return Result{}, err
	}
	holder, _, err := s.store.Identity(input.HolderDID)
	if err != nil {
		return Result{}, notFound("本地持有者身份不存在", err)
	}
	publicKey, err := holder.LSAGPublicKey()
	if err != nil {
		return Result{}, badRequest(err.Error())
	}
	bound := false
	for _, vcID := range s.store.CredentialIDsForHolder(input.HolderDID) {
		var binding protocol.MemberBinding
		if _, callErr := s.call(ctx, actorAlias, "GetMemberBinding", protocol.MemberBindingRequest{GroupID: input.GroupID, VCID: vcID}, &binding); callErr == nil && binding.Status == protocol.StatusActive && strings.EqualFold(binding.PublicKey, publicKey) {
			bound = true
			break
		}
	}
	if !bound {
		return Result{}, domain.NewError("MEMBER_BINDING_NOT_FOUND", http.StatusForbidden, "当前持有者没有有效的群组成员绑定")
	}
	if input.VPID == "" {
		input.VPID, err = randomID("pvp")
		if err != nil {
			return Result{}, err
		}
	}
	now := s.now().Unix()
	messageContext := protocol.MessageContext{
		Nonce: input.Nonce, VerifierDID: input.VerifierDID, GroupID: group.GroupID, PolicyID: group.PolicyID,
		QID: group.QID, LHash: group.LHash, TimeSlot: now / timeSlotSeconds, Purpose: input.Purpose,
	}
	signature, err := lsag.Sign(holder.LSAGPrivateScalar, group.PublicKeys, messageContext)
	if err != nil {
		return Result{}, badRequest(err.Error())
	}
	if err := lsag.Verify(group.PublicKeys, messageContext, signature); err != nil {
		return Result{}, domain.WrapError("LOCAL_LSAG_VERIFY_FAILED", http.StatusInternalServerError, "本地环签名自检失败", err)
	}
	vp := protocol.PrivacyVP{ID: input.VPID, GroupID: group.GroupID, PolicyID: group.PolicyID, QID: group.QID, LHash: group.LHash, Context: messageContext, Signature: signature}
	return Result{Data: map[string]interface{}{"vp": vp, "qualification": entry.Permission}}, nil
}

func (s *Service) VerifyPrivacyVP(ctx context.Context, actorAlias string, request protocol.VerifyPrivacyVPRequest) (Result, error) {
	actor, err := s.actor(actorAlias)
	if err != nil {
		return Result{}, err
	}
	if err := requireActorDID(actor, request.VP.Context.VerifierDID); err != nil {
		return Result{}, err
	}
	var response map[string]bool
	result, err := s.call(ctx, actorAlias, "VerifyPrivacyVP", request, &response)
	if err == nil {
		result.Data = map[string]interface{}{"verified": true, "keyImage": request.VP.Signature.KeyImage}
	}
	return result, err
}

func (s *Service) ComputeKeyImage(ctx context.Context, actorAlias string, input KeyImageInput) (Result, error) {
	actor, err := s.actor(actorAlias)
	if err != nil {
		return Result{}, err
	}
	if err := requireActorDID(actor, input.HolderDID); err != nil {
		return Result{}, err
	}
	var group protocol.CredentialGroup
	if _, err := s.call(ctx, actorAlias, "GetCredentialGroup", protocol.GroupRequest{GroupID: input.GroupID}, &group); err != nil {
		return Result{}, err
	}
	holder, _, err := s.store.Identity(input.HolderDID)
	if err != nil {
		return Result{}, notFound("本地持有者身份不存在", err)
	}
	image, err := lsag.ComputeKeyImage(holder.LSAGPrivateScalar, group.PublicKeys, group.PolicyID, group.QID, s.now().Unix()/timeSlotSeconds)
	if err != nil {
		return Result{}, badRequest(err.Error())
	}
	return Result{Data: map[string]interface{}{"keyImage": image, "groupID": group.GroupID, "policyID": group.PolicyID, "qID": group.QID, "timeSlot": s.now().Unix() / timeSlotSeconds}}, nil
}

func (s *Service) CheckKeyImage(ctx context.Context, actorAlias, image string) (Result, error) {
	var state struct {
		Used   bool                     `json:"used"`
		Record *protocol.KeyImageRecord `json:"record"`
	}
	result, err := s.call(ctx, actorAlias, "CheckKeyImage", protocol.IDRequest{ID: image}, &state)
	if err != nil {
		return Result{}, err
	}
	result.Data = state
	return result, nil
}

func (s *Service) qualificationEntry(group protocol.CredentialGroup) (govdid.PolicyEntry, error) {
	bundle, err := s.store.Policy(group.PolicyID)
	if err != nil {
		return govdid.PolicyEntry{}, notFound("本地群组策略不存在", err)
	}
	for _, entry := range bundle.Entries {
		qID, qErr := protocol.QualificationID(entry.Permission, group.IssuerDID)
		if qErr == nil && qID == group.QID {
			return entry, nil
		}
	}
	return govdid.PolicyEntry{}, domain.NewError("QUALIFICATION_NOT_FOUND", http.StatusConflict, "本地策略中找不到群组资格语义")
}
