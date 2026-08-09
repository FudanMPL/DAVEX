package govdid

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"didcontract/protocol"
	"github.com/gtank/ristretto255"
)

type Identity struct {
	DID               string `json:"did"`
	DocumentHash      string `json:"documentHash"`
	OwnerAddress      string `json:"ownerAddress"`
	P256PrivateScalar string `json:"p256PrivateScalar"`
	LSAGPrivateScalar string `json:"lsagPrivateScalar"`
	CreatedAt         int64  `json:"createdAt"`
}

func GenerateIdentity(did, documentHash, ownerAddress string) (*Identity, error) {
	did = strings.TrimSpace(did)
	ownerAddress = strings.TrimSpace(ownerAddress)
	if did == "" || ownerAddress == "" {
		return nil, errors.New("DID and owner address are required")
	}
	if _, err := protocol.DecodeHash(documentHash); err != nil {
		return nil, fmt.Errorf("document hash: %w", err)
	}
	p256, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate P-256 identity key: %w", err)
	}
	uniform := make([]byte, 64)
	if _, err := rand.Read(uniform); err != nil {
		return nil, fmt.Errorf("generate Ristretto255 identity key: %w", err)
	}
	lsag := ristretto255.NewScalar().FromUniformBytes(uniform)
	return &Identity{
		DID: did, DocumentHash: strings.ToLower(documentHash), OwnerAddress: ownerAddress,
		P256PrivateScalar: scalarHexP256(p256.D),
		LSAGPrivateScalar: hex.EncodeToString(lsag.Encode(nil)), CreatedAt: time.Now().Unix(),
	}, nil
}

func HashDocument(document string) string { return protocol.HashHex([]byte(document)) }

func (i Identity) P256PrivateKey() (*ecdsa.PrivateKey, error) {
	dBytes, err := hex.DecodeString(strings.TrimPrefix(i.P256PrivateScalar, "0x"))
	if err != nil {
		return nil, errors.New("invalid stored P-256 private scalar")
	}
	d := new(big.Int).SetBytes(dBytes)
	curve := elliptic.P256()
	if d.Sign() <= 0 || d.Cmp(curve.Params().N) >= 0 {
		return nil, errors.New("stored P-256 private scalar is out of range")
	}
	x, y := curve.ScalarBaseMult(d.Bytes())
	return &ecdsa.PrivateKey{PublicKey: ecdsa.PublicKey{Curve: curve, X: x, Y: y}, D: d}, nil
}

func (i Identity) LSAGPrivateKey() (*ristretto255.Scalar, error) {
	b, err := hex.DecodeString(strings.TrimPrefix(i.LSAGPrivateScalar, "0x"))
	if err != nil || len(b) != 32 {
		return nil, errors.New("invalid stored Ristretto255 private scalar")
	}
	privateKey := ristretto255.NewScalar()
	if err := privateKey.Decode(b); err != nil {
		return nil, errors.New("stored Ristretto255 scalar is not canonical")
	}
	return privateKey, nil
}

func (i Identity) SigningPublicKey() (string, error) {
	privateKey, err := i.P256PrivateKey()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(elliptic.Marshal(elliptic.P256(), privateKey.X, privateKey.Y)), nil
}

func (i Identity) LSAGPublicKey() (string, error) {
	privateKey, err := i.LSAGPrivateKey()
	if err != nil {
		return "", err
	}
	publicKey := ristretto255.NewElement().ScalarBaseMult(privateKey)
	return hex.EncodeToString(publicKey.Encode(nil)), nil
}

func (i Identity) Sign(message []byte) (string, error) {
	privateKey, err := i.P256PrivateKey()
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(message)
	signature, err := ecdsa.SignASN1(rand.Reader, privateKey, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign with P-256 identity key: %w", err)
	}
	return hex.EncodeToString(signature), nil
}

func (i Identity) RegisterRequest() (protocol.RegisterDIDRequest, error) {
	signingPublicKey, err := i.SigningPublicKey()
	if err != nil {
		return protocol.RegisterDIDRequest{}, err
	}
	lsagPublicKey, err := i.LSAGPublicKey()
	if err != nil {
		return protocol.RegisterDIDRequest{}, err
	}
	request := protocol.RegisterDIDRequest{
		DID: i.DID, DocumentHash: i.DocumentHash, OwnerAddress: i.OwnerAddress,
		SigningPublicKey: signingPublicKey, LSAGPublicKey: lsagPublicKey,
	}
	request.Proof, err = i.Sign(protocol.RegisterDIDMessage(request))
	return request, err
}

func (i Identity) DeactivateRequest() (protocol.DeactivateDIDRequest, error) {
	proof, err := i.Sign(protocol.DeactivateDIDMessage(i.DID))
	return protocol.DeactivateDIDRequest{DID: i.DID, Proof: proof}, err
}

func (i Identity) Rotate(newDocumentHash string) (protocol.UpdateDIDRequest, *Identity, error) {
	rotated, err := GenerateIdentity(i.DID, newDocumentHash, i.OwnerAddress)
	if err != nil {
		return protocol.UpdateDIDRequest{}, nil, err
	}
	signingPublicKey, _ := rotated.SigningPublicKey()
	lsagPublicKey, _ := rotated.LSAGPublicKey()
	request := protocol.UpdateDIDRequest{DID: i.DID, DocumentHash: newDocumentHash, SigningPublicKey: signingPublicKey, LSAGPublicKey: lsagPublicKey}
	message := protocol.UpdateDIDMessage(request)
	request.CurrentKeyProof, err = i.Sign(message)
	if err != nil {
		return protocol.UpdateDIDRequest{}, nil, err
	}
	request.NewKeyProof, err = rotated.Sign(message)
	if err != nil {
		return protocol.UpdateDIDRequest{}, nil, err
	}
	return request, rotated, nil
}

func (i Identity) SignPolicy(request protocol.RegisterPolicyRequest) (protocol.RegisterPolicyRequest, error) {
	signature, err := i.Sign(protocol.RegisterPolicyMessage(request))
	request.Signature = signature
	return request, err
}

func (i Identity) DeactivatePolicyRequest(policyID string) (protocol.DeactivatePolicyRequest, error) {
	signature, err := i.Sign(protocol.DeactivatePolicyMessage(policyID))
	return protocol.DeactivatePolicyRequest{PolicyID: policyID, Signature: signature}, err
}

func (i Identity) SignCredential(credential protocol.VerifiableCredential) (protocol.VerifiableCredential, error) {
	message, err := protocol.CredentialBytes(credential)
	if err != nil {
		return protocol.VerifiableCredential{}, err
	}
	credential.Signature, err = i.Sign(message)
	return credential, err
}

func (i Identity) SignPresentation(presentation protocol.StandardVP) (protocol.StandardVP, error) {
	message, err := protocol.VPBytes(presentation)
	if err != nil {
		return protocol.StandardVP{}, err
	}
	presentation.Signature, err = i.Sign(message)
	return presentation, err
}

func (i Identity) SignRevocationCredential(credential protocol.RevocationCredential) (protocol.RevocationCredential, error) {
	credential.Signature = ""
	signature, err := i.Sign(protocol.RevocationCredentialBytes(credential))
	credential.Signature = signature
	return credential, err
}

func scalarHexP256(value *big.Int) string {
	return fmt.Sprintf("%064x", value)
}

func DeterministicLSAGScalar(seed string) *ristretto255.Scalar {
	sum := sha512.Sum512([]byte(seed))
	return ristretto255.NewScalar().FromUniformBytes(sum[:])
}
