package main

const (
	prefixSystem        = "system"
	prefixDID           = "did_registry"
	prefixAddress       = "did_address"
	prefixRole          = "role_binding"
	prefixPolicy        = "policy_anchor"
	prefixVC            = "vc_proof"
	prefixNonce         = "nonce"
	prefixGroup         = "credential_group"
	prefixBinding       = "member_binding"
	prefixBindingPub    = "member_binding_pub"
	prefixKeyImage      = "used_key_image"
	prefixEventIssuer   = "event_issuer"
	prefixCommittee     = "revocation_committee"
	prefixConsumed      = "consumed_credential"
	prefixRevocationLog = "revocation_log"
	prefixRevocationIdx = "revocation_log_index"
	prefixRotationLog   = "committee_rotation_log"
	prefixRotationIdx   = "committee_rotation_index"
)

const (
	maxIDLength            = 256
	maxRingSize            = 200
	maxCommitteeSize       = 32
	maxProofDepth          = 64
	timeSlotSeconds  int64 = 300
)

const (
	topicDIDRegistered      = "DIDRegistered"
	topicDIDUpdated         = "DIDUpdated"
	topicDIDDeactivated     = "DIDDeactivated"
	topicPolicyRegistered   = "PolicyRegistered"
	topicPolicyDeactivated  = "PolicyDeactivated"
	topicVCIssued           = "VCIssued"
	topicNonceIssued        = "NonceIssued"
	topicVPVerified         = "VPVerified"
	topicGroupCreated       = "CredentialGroupCreated"
	topicGroupMemberAdded   = "CredentialGroupMemberAdded"
	topicGroupStatusChanged = "CredentialGroupStatusChanged"
	topicPrivacyVPVerified  = "PrivacyVPVerified"
	topicGovernanceBound    = "GovernanceDIDBound"
	topicEventIssuerUpdated = "EventIssuerUpdated"
	topicCommitteeUpdated   = "RevocationCommitteeUpdated"
	topicCredentialRevoked  = "CredentialRevoked"
)

type SystemConfig struct {
	BootstrapAddress string `json:"bootstrapAddress"`
	GovernanceDID    string `json:"governanceDID,omitempty"`
}
