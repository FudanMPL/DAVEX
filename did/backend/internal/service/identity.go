package service

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"didcontract/backend/internal/domain"
	"didcontract/protocol"
	"didcontract/sdk-go/govdid"
)

type GenerateDIDInput struct {
	DID      string `json:"did"`
	Document string `json:"document"`
}

type RegisterDIDInput struct {
	DID string `json:"did"`
}

type RotateDIDInput struct {
	DID      string `json:"did"`
	Document string `json:"document"`
}

type PolicyInput struct {
	IssuerDID   string                `json:"issuerDID"`
	Permissions []protocol.Permission `json:"permissions"`
}

type IssueVCInput struct {
	VCID              string `json:"vcID"`
	HolderDID         string `json:"holderDID"`
	IssuerDID         string `json:"issuerDID"`
	PolicyID          string `json:"policyID"`
	EntryIndex        int    `json:"entryIndex"`
	ExpiresAt         int64  `json:"expiresAt"`
	AnonymousEligible bool   `json:"anonymousEligible"`
}

type IssueNonceInput struct {
	VerifierDID string `json:"verifierDID"`
	Purpose     string `json:"purpose"`
	TTLSeconds  int64  `json:"ttlSeconds"`
}

type GenerateVPInput struct {
	HolderDID   string   `json:"holderDID"`
	VerifierDID string   `json:"verifierDID"`
	Nonce       string   `json:"nonce"`
	Purpose     string   `json:"purpose"`
	VCIDs       []string `json:"vcIDs"`
}

type VerifyVPInput struct {
	VP     protocol.StandardVP     `json:"vp"`
	Access *protocol.AccessRequest `json:"access,omitempty"`
}

func (s *Service) GenerateDID(actorAlias string, input GenerateDIDInput) (Result, error) {
	actor, err := s.actor(actorAlias)
	if err != nil {
		return Result{}, err
	}
	input.DID = strings.TrimSpace(input.DID)
	if err := requireActorDID(actor, input.DID); err != nil {
		return Result{}, err
	}
	if strings.TrimSpace(input.Document) == "" {
		return Result{}, badRequest("DID 文档不能为空")
	}
	for _, did := range s.store.IdentityDIDs() {
		existing, _, loadErr := s.store.Identity(did)
		if loadErr == nil && existing.OwnerAddress == actor.Origin {
			return Result{}, conflict("当前 ChainMaker 交易身份已关联本地 DID "+did, nil)
		}
	}
	identity, err := govdid.GenerateIdentity(input.DID, govdid.HashDocument(input.Document), actor.Origin)
	if err != nil {
		return Result{}, badRequest(err.Error())
	}
	if err := s.store.PutIdentity(*identity, input.Document); err != nil {
		return Result{}, conflict("本地 DID 已存在", err)
	}
	signingKey, _ := identity.SigningPublicKey()
	lsagKey, _ := identity.LSAGPublicKey()
	return Result{Data: map[string]interface{}{
		"did": input.DID, "documentHash": identity.DocumentHash, "ownerAddress": actor.Origin,
		"signingPublicKey": signingKey, "lsagPublicKey": lsagKey, "keyRef": input.DID,
	}}, nil
}

func (s *Service) RegisterDID(ctx context.Context, actorAlias string, input RegisterDIDInput) (Result, error) {
	actor, err := s.actor(actorAlias)
	if err != nil {
		return Result{}, err
	}
	if err := requireActorDID(actor, input.DID); err != nil {
		return Result{}, err
	}
	identity, _, err := s.store.Identity(input.DID)
	if err != nil {
		return Result{}, notFound("本地 DID 不存在", err)
	}
	if identity.OwnerAddress != actor.Origin {
		return Result{}, domain.NewError("OWNER_MISMATCH", http.StatusForbidden, "DID owner 与当前交易身份不一致")
	}
	request, err := identity.RegisterRequest()
	if err != nil {
		return Result{}, badRequest(err.Error())
	}
	var registry protocol.DIDRegistry
	return s.call(ctx, actorAlias, "RegisterDID", request, &registry)
}

func (s *Service) QueryDID(ctx context.Context, actorAlias, did string) (Result, error) {
	var registry protocol.DIDRegistry
	result, err := s.call(ctx, actorAlias, "GetDIDRegistry", protocol.IDRequest{ID: did}, &registry)
	if err != nil {
		return Result{}, err
	}
	_, document, localErr := s.store.Identity(did)
	result.Data = map[string]interface{}{"registry": registry, "document": document, "localDocumentAvailable": localErr == nil}
	return result, nil
}

func (s *Service) ListDIDs() Result { return Result{Data: s.store.IdentityDIDs()} }

func (s *Service) RotateDID(ctx context.Context, actorAlias string, input RotateDIDInput) (Result, error) {
	actor, err := s.actor(actorAlias)
	if err != nil {
		return Result{}, err
	}
	if err := requireActorDID(actor, input.DID); err != nil {
		return Result{}, err
	}
	identity, _, err := s.store.Identity(input.DID)
	if err != nil {
		return Result{}, notFound("本地 DID 不存在", err)
	}
	request, rotated, err := identity.Rotate(govdid.HashDocument(input.Document))
	if err != nil {
		return Result{}, badRequest(err.Error())
	}
	var registry protocol.DIDRegistry
	result, err := s.call(ctx, actorAlias, "UpdateDID", request, &registry)
	if err != nil {
		return Result{}, err
	}
	if err := s.store.ReplaceIdentity(*rotated, input.Document); err != nil {
		return Result{}, conflict("链上换钥成功但本地密钥替换失败", err)
	}
	return result, nil
}

func (s *Service) DeactivateDID(ctx context.Context, actorAlias, did string) (Result, error) {
	actor, err := s.actor(actorAlias)
	if err != nil {
		return Result{}, err
	}
	if err := requireActorDID(actor, did); err != nil {
		return Result{}, err
	}
	identity, _, err := s.store.Identity(did)
	if err != nil {
		return Result{}, notFound("本地 DID 不存在", err)
	}
	request, err := identity.DeactivateRequest()
	if err != nil {
		return Result{}, badRequest(err.Error())
	}
	var response map[string]bool
	return s.call(ctx, actorAlias, "DeactivateDID", request, &response)
}

func (s *Service) SetGovernance(ctx context.Context, actorAlias, did string) (Result, error) {
	var response map[string]bool
	return s.call(ctx, actorAlias, "SetGovernanceDID", protocol.SetGovernanceDIDRequest{DID: did}, &response)
}

func (s *Service) UpdateRoles(ctx context.Context, actorAlias string, request protocol.UpdateRolesRequest) (Result, error) {
	var binding protocol.RoleBinding
	return s.call(ctx, actorAlias, "UpdateRoles", request, &binding)
}

func (s *Service) GetRoles(ctx context.Context, actorAlias, did string) (Result, error) {
	var binding protocol.RoleBinding
	return s.call(ctx, actorAlias, "GetRoles", protocol.IDRequest{ID: did}, &binding)
}

func (s *Service) RegisterPolicy(ctx context.Context, actorAlias string, input PolicyInput) (Result, error) {
	actor, err := s.actor(actorAlias)
	if err != nil {
		return Result{}, err
	}
	if err := requireActorDID(actor, input.IssuerDID); err != nil {
		return Result{}, err
	}
	bundle, err := govdid.BuildPolicy(input.IssuerDID, input.Permissions)
	if err != nil {
		return Result{}, badRequest(err.Error())
	}
	issuer, _, err := s.store.Identity(input.IssuerDID)
	if err != nil {
		return Result{}, notFound("本地策略签发方不存在", err)
	}
	request, err := issuer.SignPolicy(bundle.RegistrationRequest())
	if err != nil {
		return Result{}, badRequest(err.Error())
	}
	var anchor protocol.PolicyAnchor
	result, err := s.call(ctx, actorAlias, "RegisterPolicy", request, &anchor)
	if err != nil {
		return Result{}, err
	}
	if err := s.store.PutPolicy(*bundle); err != nil {
		return Result{}, conflict("策略已上链但本地语义保存失败", err)
	}
	result.Data = map[string]interface{}{"anchor": anchor, "bundle": bundle}
	return result, nil
}

func (s *Service) QueryPolicy(ctx context.Context, actorAlias, policyID string) (Result, error) {
	var anchor protocol.PolicyAnchor
	result, err := s.call(ctx, actorAlias, "GetPolicy", protocol.IDRequest{ID: policyID}, &anchor)
	if err != nil {
		return Result{}, err
	}
	bundle, localErr := s.store.Policy(policyID)
	result.Data = map[string]interface{}{"anchor": anchor, "bundle": bundle, "localSemanticsAvailable": localErr == nil}
	return result, nil
}

func (s *Service) DeactivatePolicy(ctx context.Context, actorAlias, policyID string) (Result, error) {
	bundle, err := s.store.Policy(policyID)
	if err != nil {
		return Result{}, notFound("本地策略不存在", err)
	}
	actor, err := s.actor(actorAlias)
	if err != nil {
		return Result{}, err
	}
	if err := requireActorDID(actor, bundle.IssuerDID); err != nil {
		return Result{}, err
	}
	identity, _, err := s.store.Identity(bundle.IssuerDID)
	if err != nil {
		return Result{}, notFound("本地策略签发方不存在", err)
	}
	request, err := identity.DeactivatePolicyRequest(policyID)
	if err != nil {
		return Result{}, badRequest(err.Error())
	}
	var anchor protocol.PolicyAnchor
	return s.call(ctx, actorAlias, "DeactivatePolicy", request, &anchor)
}

func (s *Service) VerifyPolicy(ctx context.Context, actorAlias string, request protocol.VerifyPolicyMembershipRequest) (Result, error) {
	if len(request.Proof) == 0 {
		bundle, err := s.store.Policy(request.PolicyID)
		if err != nil {
			return Result{}, notFound("本地策略证明不存在", err)
		}
		for _, entry := range bundle.Entries {
			if permissionsEqual(entry.Permission, request.Permission) {
				for _, proof := range entry.Proofs {
					if proof.Action == request.Action {
						request.Proof = proof.Steps
					}
				}
			}
		}
	}
	var response map[string]bool
	return s.call(ctx, actorAlias, "VerifyPolicyMembership", request, &response)
}

func (s *Service) IssueVC(ctx context.Context, actorAlias string, input IssueVCInput) (Result, error) {
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
		return Result{}, badRequest("凭证签发方与策略签发方不一致")
	}
	entry, err := bundle.Entry(input.EntryIndex)
	if err != nil {
		return Result{}, badRequest(err.Error())
	}
	now := s.now().Unix()
	if now < entry.Permission.ValidFrom || now >= entry.Permission.ValidUntil {
		return Result{}, badRequest("当前时间不在权限有效期内")
	}
	if input.ExpiresAt == 0 || input.ExpiresAt > entry.Permission.ValidUntil {
		input.ExpiresAt = entry.Permission.ValidUntil
	}
	if input.ExpiresAt <= now {
		return Result{}, badRequest("凭证过期时间必须晚于当前时间")
	}
	issuer, _, err := s.store.Identity(input.IssuerDID)
	if err != nil {
		return Result{}, notFound("本地签发方身份不存在", err)
	}
	credential := protocol.VerifiableCredential{
		ID: input.VCID, HolderDID: input.HolderDID, IssuerDID: input.IssuerDID, PolicyID: input.PolicyID,
		Permission: entry.Permission, IssuedAt: now, ExpiresAt: input.ExpiresAt, AnonymousEligible: input.AnonymousEligible,
	}
	credential, err = issuer.SignCredential(credential)
	if err != nil {
		return Result{}, badRequest(err.Error())
	}
	request := protocol.IssueVCRequest{Credential: credential, Proofs: entry.Proofs}
	var proof protocol.VCProof
	result, err := s.call(ctx, actorAlias, "IssueVC", request, &proof)
	if err != nil {
		return Result{}, err
	}
	if err := s.store.PutCredential(request); err != nil {
		return Result{}, conflict("VCProof 已上链但完整 VC 保存失败", err)
	}
	result.Data = map[string]interface{}{"credential": credential, "proof": proof}
	return result, nil
}

func (s *Service) QueryVC(ctx context.Context, actorAlias, vcID string) (Result, error) {
	var proof protocol.VCProof
	result, err := s.call(ctx, actorAlias, "GetVCProof", protocol.IDRequest{ID: vcID}, &proof)
	if err != nil {
		return Result{}, err
	}
	credential, localErr := s.store.Credential(vcID)
	actor, actorErr := s.actor(actorAlias)
	allowed := localErr == nil && actorErr == nil && (actor.DID == credential.Credential.HolderDID || actor.DID == credential.Credential.IssuerDID)
	data := map[string]interface{}{"proof": proof, "localCredentialAvailable": localErr == nil, "fullCredentialVisible": allowed}
	if allowed {
		data["credential"] = credential.Credential
	}
	result.Data = data
	return result, nil
}

func (s *Service) VerifyVC(ctx context.Context, actorAlias, vcID string) (Result, error) {
	request, err := s.store.Credential(vcID)
	if err != nil {
		return Result{}, notFound("本地完整 VC 不存在", err)
	}
	var proof protocol.VCProof
	return s.call(ctx, actorAlias, "VerifyVC", request, &proof)
}

func (s *Service) IssueNonce(ctx context.Context, actorAlias string, input IssueNonceInput) (Result, error) {
	actor, err := s.actor(actorAlias)
	if err != nil {
		return Result{}, err
	}
	if err := requireActorDID(actor, input.VerifierDID); err != nil {
		return Result{}, err
	}
	if input.TTLSeconds <= 0 || input.TTLSeconds > 600 {
		input.TTLSeconds = 120
	}
	nonce, err := randomID("nonce")
	if err != nil {
		return Result{}, err
	}
	now := s.now().Unix()
	request := protocol.IssueNonceRequest{VerifierDID: input.VerifierDID, Nonce: nonce, Purpose: input.Purpose, ExpiresAt: now + input.TTLSeconds}
	var record protocol.NonceRecord
	result, err := s.call(ctx, actorAlias, "IssueNonce", request, &record)
	if err != nil {
		return Result{}, err
	}
	_ = s.store.PutSession(domain.VerificationSession{Nonce: nonce, VerifierDID: input.VerifierDID, Purpose: input.Purpose, ExpiresAt: request.ExpiresAt, Status: protocol.StatusActive, CreatedAt: now})
	return result, nil
}

func (s *Service) GenerateVP(actorAlias string, input GenerateVPInput) (Result, error) {
	actor, err := s.actor(actorAlias)
	if err != nil {
		return Result{}, err
	}
	if err := requireActorDID(actor, input.HolderDID); err != nil {
		return Result{}, err
	}
	if len(input.VCIDs) == 0 {
		return Result{}, badRequest("至少选择一张 VC")
	}
	presentations := make([]protocol.CredentialPresentation, 0, len(input.VCIDs))
	credentials := make([]protocol.VerifiableCredential, 0, len(input.VCIDs))
	for _, vcID := range input.VCIDs {
		stored, err := s.store.Credential(vcID)
		if err != nil {
			return Result{}, notFound("本地完整 VC 不存在", err)
		}
		if stored.Credential.HolderDID != input.HolderDID {
			return Result{}, badRequest("VC 持有者与当前 DID 不一致")
		}
		credentials = append(credentials, stored.Credential)
		presentations = append(presentations, protocol.CredentialPresentation{Credential: stored.Credential, Proofs: stored.Proofs})
	}
	holder, _, err := s.store.Identity(input.HolderDID)
	if err != nil {
		return Result{}, notFound("本地持有者身份不存在", err)
	}
	vp := protocol.StandardVP{HolderDID: input.HolderDID, VerifierDID: input.VerifierDID, Nonce: input.Nonce, Purpose: input.Purpose, Credentials: credentials}
	vp, err = holder.SignPresentation(vp)
	if err != nil {
		return Result{}, badRequest(err.Error())
	}
	return Result{Data: map[string]interface{}{"vp": vp, "presentations": presentations}}, nil
}

func (s *Service) VerifyVP(ctx context.Context, actorAlias string, input VerifyVPInput) (Result, error) {
	actor, err := s.actor(actorAlias)
	if err != nil {
		return Result{}, err
	}
	if err := requireActorDID(actor, input.VP.VerifierDID); err != nil {
		return Result{}, err
	}
	presentations := make([]protocol.CredentialPresentation, 0, len(input.VP.Credentials))
	for _, credential := range input.VP.Credentials {
		stored, err := s.store.Credential(credential.ID)
		if err != nil {
			return Result{}, notFound("本地完整 VC 或证明不存在", err)
		}
		presentations = append(presentations, protocol.CredentialPresentation{Credential: credential, Proofs: stored.Proofs})
	}
	request := protocol.VerifyVPRequest{VP: input.VP, Presentations: presentations, Access: input.Access}
	var response map[string]bool
	result, err := s.call(ctx, actorAlias, "VerifyVP", request, &response)
	if err == nil {
		result.Data = map[string]interface{}{"verified": true, "holderDID": input.VP.HolderDID, "access": input.Access}
	}
	return result, err
}

func permissionsEqual(a, b protocol.Permission) bool {
	if a.PolicyID != b.PolicyID || a.DeptRole != b.DeptRole || a.AuthScope != b.AuthScope || a.DataLevel != b.DataLevel || a.ValidFrom != b.ValidFrom || a.ValidUntil != b.ValidUntil {
		return false
	}
	actionsA := append([]string(nil), a.ActionSet...)
	actionsB := append([]string(nil), b.ActionSet...)
	sort.Strings(actionsA)
	sort.Strings(actionsB)
	return fmt.Sprint(actionsA) == fmt.Sprint(actionsB)
}
