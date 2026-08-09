package protocol

const (
	StatusActive   = "Active"
	StatusInactive = "Inactive"
	StatusValid    = "Valid"
	StatusRevoked  = "Revoked"
	StatusUsed     = "Used"

	ScopeGroup  = "group"
	ScopePolicy = "policy"
)

// Permission is the normalized six-tuple permission carried by a VC.
type Permission struct {
	PolicyID   string   `json:"policyID"`
	DeptRole   string   `json:"deptRole"`
	AuthScope  string   `json:"authScope"`
	DataLevel  string   `json:"dataLevel"`
	ActionSet  []string `json:"actionSet"`
	ValidFrom  int64    `json:"validFrom"`
	ValidUntil int64    `json:"validUntil"`
}

type AccessRequest struct {
	DeptRole  string `json:"deptRole"`
	AuthScope string `json:"authScope"`
	DataLevel string `json:"dataLevel"`
	Action    string `json:"action"`
	AtTime    int64  `json:"atTime"`
}

type MerkleStep struct {
	Sibling  string `json:"sibling"`
	Position string `json:"position"`
}

type ActionProof struct {
	Action string       `json:"action"`
	Steps  []MerkleStep `json:"steps"`
}

type VerifyPolicyMembershipRequest struct {
	PolicyID   string       `json:"policyID"`
	IssuerDID  string       `json:"issuerDID"`
	Permission Permission   `json:"permission"`
	Action     string       `json:"action"`
	Proof      []MerkleStep `json:"proof"`
}

type DIDRegistry struct {
	DID              string `json:"did"`
	DocumentHash     string `json:"documentHash"`
	OwnerAddress     string `json:"ownerAddress"`
	SigningPublicKey string `json:"signingPublicKey"`
	LSAGPublicKey    string `json:"lsagPublicKey"`
	Status           string `json:"status"`
	CreatedAt        int64  `json:"createdAt"`
	UpdatedAt        int64  `json:"updatedAt"`
}

type RoleBinding struct {
	DID       string   `json:"did"`
	Roles     []string `json:"roles"`
	UpdatedAt int64    `json:"updatedAt"`
}

type UpdateRolesRequest struct {
	DID   string   `json:"did"`
	Roles []string `json:"roles"`
}

type RegisterDIDRequest struct {
	DID              string `json:"did"`
	DocumentHash     string `json:"documentHash"`
	OwnerAddress     string `json:"ownerAddress"`
	SigningPublicKey string `json:"signingPublicKey"`
	LSAGPublicKey    string `json:"lsagPublicKey"`
	Proof            string `json:"proof"`
}

type UpdateDIDRequest struct {
	DID              string `json:"did"`
	DocumentHash     string `json:"documentHash"`
	SigningPublicKey string `json:"signingPublicKey"`
	LSAGPublicKey    string `json:"lsagPublicKey"`
	CurrentKeyProof  string `json:"currentKeyProof"`
	NewKeyProof      string `json:"newKeyProof"`
}

type DeactivateDIDRequest struct {
	DID   string `json:"did"`
	Proof string `json:"proof"`
}

type PolicyAnchor struct {
	PolicyID   string `json:"policyID"`
	PolicyHash string `json:"policyHash"`
	IssuerDID  string `json:"issuerDID"`
	RuleCount  int    `json:"ruleCount"`
	Status     string `json:"status"`
	CreatedAt  int64  `json:"createdAt"`
	UpdatedAt  int64  `json:"updatedAt"`
	Signature  string `json:"signature"`
}

type RegisterPolicyRequest struct {
	PolicyID   string `json:"policyID"`
	PolicyHash string `json:"policyHash"`
	IssuerDID  string `json:"issuerDID"`
	RuleCount  int    `json:"ruleCount"`
	Signature  string `json:"signature"`
}

type DeactivatePolicyRequest struct {
	PolicyID  string `json:"policyID"`
	Signature string `json:"signature"`
}

type VerifiableCredential struct {
	ID                string     `json:"id"`
	HolderDID         string     `json:"holderDID"`
	IssuerDID         string     `json:"issuerDID"`
	PolicyID          string     `json:"policyID"`
	Permission        Permission `json:"permission"`
	IssuedAt          int64      `json:"issuedAt"`
	ExpiresAt         int64      `json:"expiresAt"`
	AnonymousEligible bool       `json:"anonymousEligible"`
	Signature         string     `json:"signature"`
}

type IssueVCRequest struct {
	Credential VerifiableCredential `json:"credential"`
	Proofs     []ActionProof        `json:"proofs"`
}

type VCProof struct {
	VCID              string `json:"vcID"`
	CredentialHash    string `json:"credentialHash"`
	HolderDID         string `json:"holderDID"`
	IssuerDID         string `json:"issuerDID"`
	PolicyID          string `json:"policyID"`
	Status            string `json:"status"`
	IssuedAt          int64  `json:"issuedAt"`
	ExpiresAt         int64  `json:"expiresAt"`
	AnonymousEligible bool   `json:"anonymousEligible"`
}

type StandardVP struct {
	HolderDID   string                 `json:"holderDID"`
	VerifierDID string                 `json:"verifierDID"`
	Nonce       string                 `json:"nonce"`
	Purpose     string                 `json:"purpose"`
	Credentials []VerifiableCredential `json:"credentials"`
	Signature   string                 `json:"signature"`
}

type CredentialPresentation struct {
	Credential VerifiableCredential `json:"credential"`
	Proofs     []ActionProof        `json:"proofs"`
}

type VerifyVPRequest struct {
	VP            StandardVP               `json:"vp"`
	Presentations []CredentialPresentation `json:"presentations"`
	Access        *AccessRequest           `json:"access,omitempty"`
}

type NonceRecord struct {
	VerifierDID string `json:"verifierDID"`
	Nonce       string `json:"nonce"`
	Purpose     string `json:"purpose"`
	ExpiresAt   int64  `json:"expiresAt"`
	Status      string `json:"status"`
}

type IssueNonceRequest struct {
	VerifierDID string `json:"verifierDID"`
	Nonce       string `json:"nonce"`
	Purpose     string `json:"purpose"`
	ExpiresAt   int64  `json:"expiresAt"`
}

type CredentialGroup struct {
	GroupID     string   `json:"groupID"`
	PolicyID    string   `json:"policyID"`
	QID         string   `json:"qID"`
	IssuerDID   string   `json:"issuerDID"`
	PublicKeys  []string `json:"publicKeys"`
	MemberEpoch uint64   `json:"memberEpoch"`
	LHash       string   `json:"lHash"`
	Status      string   `json:"status"`
	CreatedAt   int64    `json:"createdAt"`
	UpdatedAt   int64    `json:"updatedAt"`
}

type CreateGroupRequest struct {
	GroupID       string        `json:"groupID"`
	PolicyID      string        `json:"policyID"`
	IssuerDID     string        `json:"issuerDID"`
	Qualification Permission    `json:"qualification"`
	Proofs        []ActionProof `json:"proofs"`
}

type AddGroupMemberRequest struct {
	GroupID    string               `json:"groupID"`
	Credential VerifiableCredential `json:"credential"`
	Proofs     []ActionProof        `json:"proofs"`
	PublicKey  string               `json:"publicKey"`
}

type MemberBinding struct {
	GroupID   string `json:"groupID"`
	VCID      string `json:"vcID"`
	HolderDID string `json:"holderDID"`
	PublicKey string `json:"publicKey"`
	Status    string `json:"status"`
	JoinedAt  int64  `json:"joinedAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

type MessageContext struct {
	Nonce       string `json:"nonce"`
	VerifierDID string `json:"verifierDID"`
	GroupID     string `json:"groupID"`
	PolicyID    string `json:"policyID"`
	QID         string `json:"qID"`
	LHash       string `json:"lHash"`
	TimeSlot    int64  `json:"timeSlot"`
	Purpose     string `json:"purpose"`
}

type LSAGSignature struct {
	Algorithm string   `json:"algorithm"`
	C0        string   `json:"c0"`
	Responses []string `json:"responses"`
	KeyImage  string   `json:"keyImage"`
}

type PrivacyVP struct {
	ID        string         `json:"id"`
	GroupID   string         `json:"groupID"`
	PolicyID  string         `json:"policyID"`
	QID       string         `json:"qID"`
	LHash     string         `json:"lHash"`
	Context   MessageContext `json:"context"`
	Signature LSAGSignature  `json:"signature"`
}

type VerifyPrivacyVPRequest struct {
	VP            PrivacyVP     `json:"vp"`
	Qualification Permission    `json:"qualification"`
	Access        AccessRequest `json:"access"`
}

type KeyImageRecord struct {
	KeyImage string `json:"keyImage"`
	GroupID  string `json:"groupID"`
	PolicyID string `json:"policyID"`
	QID      string `json:"qID"`
	TimeSlot int64  `json:"timeSlot"`
	UsedAt   int64  `json:"usedAt"`
	TxID     string `json:"txID"`
}

type EventIssuerEntry struct {
	EventType     string `json:"eventType"`
	IssuerDID     string `json:"issuerDID"`
	MaxAgeSeconds int64  `json:"maxAgeSeconds"`
	Status        string `json:"status"`
	UpdatedAt     int64  `json:"updatedAt"`
}

type RegisterEventIssuerRequest struct {
	EventType     string `json:"eventType"`
	IssuerDID     string `json:"issuerDID"`
	MaxAgeSeconds int64  `json:"maxAgeSeconds"`
}

type CommitteeMember struct {
	DID       string `json:"did"`
	PublicKey string `json:"publicKey"`
	TermStart int64  `json:"termStart"`
	TermEnd   int64  `json:"termEnd"`
	Status    string `json:"status"`
}

type RevocationCommittee struct {
	GroupID   string            `json:"groupID"`
	IssuerDID string            `json:"issuerDID"`
	Threshold int               `json:"threshold"`
	Members   []CommitteeMember `json:"members"`
	Epoch     uint64            `json:"epoch"`
	Status    string            `json:"status"`
	CreatedAt int64             `json:"createdAt"`
	UpdatedAt int64             `json:"updatedAt"`
}

type CreateCommitteeRequest struct {
	GroupID   string            `json:"groupID"`
	Threshold int               `json:"threshold"`
	Members   []CommitteeMember `json:"members"`
}

type CommitteeMemberRequest struct {
	GroupID string          `json:"groupID"`
	Member  CommitteeMember `json:"member"`
}

type UpdateThresholdRequest struct {
	GroupID   string `json:"groupID"`
	Threshold int    `json:"threshold"`
}

type RevocationCredential struct {
	ID         string `json:"id"`
	IssuerDID  string `json:"issuerDID"`
	SubjectDID string `json:"subjectDID"`
	TargetVCID string `json:"targetVCID"`
	EventType  string `json:"eventType"`
	EventTime  int64  `json:"eventTime"`
	ScopeType  string `json:"scopeType"`
	ScopeID    string `json:"scopeID"`
	Signature  string `json:"signature"`
}

type PartialSignature struct {
	MemberDID string `json:"memberDID"`
	R         string `json:"r"`
	S         string `json:"s"`
}

type ExecuteRevocationRequest struct {
	Credential      RevocationCredential `json:"credential"`
	GroupID         string               `json:"groupID"`
	HolderDID       string               `json:"holderDID"`
	HolderPublicKey string               `json:"holderPublicKey"`
	VCID            string               `json:"vcID"`
	Nonce           string               `json:"nonce"`
	Approvals       []PartialSignature   `json:"approvals"`
}

type RevocationLog struct {
	ID             string   `json:"id"`
	CredentialHash string   `json:"credentialHash"`
	CredentialID   string   `json:"credentialID"`
	GroupID        string   `json:"groupID"`
	VCID           string   `json:"vcID"`
	HolderDID      string   `json:"holderDID"`
	EventType      string   `json:"eventType"`
	EventTime      int64    `json:"eventTime"`
	Approvers      []string `json:"approvers"`
	ExecutedAt     int64    `json:"executedAt"`
	TxID           string   `json:"txID"`
}

type RotationLog struct {
	ID        string `json:"id"`
	GroupID   string `json:"groupID"`
	Operation string `json:"operation"`
	MemberDID string `json:"memberDID,omitempty"`
	Threshold int    `json:"threshold"`
	Epoch     uint64 `json:"epoch"`
	Operator  string `json:"operator"`
	CreatedAt int64  `json:"createdAt"`
}

type IDRequest struct {
	ID string `json:"id"`
}

type GroupRequest struct {
	GroupID string `json:"groupID"`
}

type MemberBindingRequest struct {
	GroupID string `json:"groupID"`
	VCID    string `json:"vcID"`
}

type SetGovernanceDIDRequest struct {
	DID string `json:"did"`
}
