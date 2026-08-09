package revocation

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"didcontract/protocol"
	"didcontract/sdk-go/govdid"
	"github.com/gtank/ristretto255"
)

// VerifyApproval checks one committee Schnorr approval before it is added to a
// draft. The contract remains the final authority during ExecuteRevocation.
func VerifyApproval(publicKeyHex string, approval protocol.PartialSignature, credential protocol.RevocationCredential, groupID, holderDID, vcID, nonce string) error {
	publicKey, err := decodePoint(publicKeyHex)
	if err != nil {
		return fmt.Errorf("committee public key: %w", err)
	}
	commitment, err := decodePoint(approval.R)
	if err != nil {
		return fmt.Errorf("approval commitment: %w", err)
	}
	response, err := decodeScalar(approval.S)
	if err != nil {
		return fmt.Errorf("approval response: %w", err)
	}
	message := protocol.RevocationMessage(protocol.RevocationCredentialHash(credential), groupID, holderDID, vcID, nonce)
	challenge := schnorrChallenge(publicKey.Encode(nil), commitment.Encode(nil), message)
	left := ristretto255.NewElement().ScalarBaseMult(response)
	right := ristretto255.NewElement().Add(commitment, ristretto255.NewElement().ScalarMult(challenge, publicKey))
	if left.Equal(right) != 1 {
		return errors.New("Schnorr approval equation does not hold")
	}
	return nil
}

func decodePoint(value string) (*ristretto255.Element, error) {
	data, err := hex.DecodeString(stripHex(value))
	if err != nil || len(data) != 32 {
		return nil, errors.New("invalid Ristretto255 point encoding")
	}
	point := ristretto255.NewElement()
	if err := point.Decode(data); err != nil || point.Equal(ristretto255.NewElement().Zero()) == 1 {
		return nil, errors.New("invalid Ristretto255 point")
	}
	return point, nil
}

func decodeScalar(value string) (*ristretto255.Scalar, error) {
	data, err := hex.DecodeString(stripHex(value))
	if err != nil || len(data) != 32 {
		return nil, errors.New("invalid Ristretto255 scalar encoding")
	}
	scalar := ristretto255.NewScalar()
	if err := scalar.Decode(data); err != nil {
		return nil, errors.New("non-canonical Ristretto255 scalar")
	}
	return scalar, nil
}

func stripHex(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "0x")
	return strings.TrimPrefix(value, "0X")
}

func BuildCredential(eventIssuer govdid.Identity, credential protocol.RevocationCredential) (protocol.RevocationCredential, error) {
	if credential.IssuerDID != eventIssuer.DID {
		return protocol.RevocationCredential{}, errors.New("event issuer DID does not match signing identity")
	}
	return eventIssuer.SignRevocationCredential(credential)
}

func Approve(member govdid.Identity, credential protocol.RevocationCredential, groupID, holderDID, vcID, nonce string) (protocol.PartialSignature, error) {
	privateKey, err := member.LSAGPrivateKey()
	if err != nil {
		return protocol.PartialSignature{}, err
	}
	publicKey := ristretto255.NewElement().ScalarBaseMult(privateKey)
	message := protocol.RevocationMessage(protocol.RevocationCredentialHash(credential), groupID, holderDID, vcID, nonce)
	nonceScalar, err := randomScalar()
	if err != nil {
		return protocol.PartialSignature{}, err
	}
	commitment := ristretto255.NewElement().ScalarBaseMult(nonceScalar)
	challenge := schnorrChallenge(publicKey.Encode(nil), commitment.Encode(nil), message)
	product := ristretto255.NewScalar().Multiply(challenge, privateKey)
	response := ristretto255.NewScalar().Add(nonceScalar, product)
	return protocol.PartialSignature{
		MemberDID: member.DID,
		R:         hex.EncodeToString(commitment.Encode(nil)),
		S:         hex.EncodeToString(response.Encode(nil)),
	}, nil
}

func BuildExecuteRequest(
	credential protocol.RevocationCredential,
	groupID, holderDID, holderPublicKey, vcID, nonce string,
	members []govdid.Identity,
) (protocol.ExecuteRevocationRequest, error) {
	if len(members) == 0 {
		return protocol.ExecuteRevocationRequest{}, errors.New("at least one committee approval is required")
	}
	approvals := make([]protocol.PartialSignature, 0, len(members))
	seen := make(map[string]struct{}, len(members))
	for _, member := range members {
		if _, exists := seen[member.DID]; exists {
			return protocol.ExecuteRevocationRequest{}, fmt.Errorf("duplicate committee identity %s", member.DID)
		}
		seen[member.DID] = struct{}{}
		approval, err := Approve(member, credential, groupID, holderDID, vcID, nonce)
		if err != nil {
			return protocol.ExecuteRevocationRequest{}, err
		}
		approvals = append(approvals, approval)
	}
	return protocol.ExecuteRevocationRequest{
		Credential: credential, GroupID: groupID, HolderDID: holderDID,
		HolderPublicKey: holderPublicKey, VCID: vcID, Nonce: nonce, Approvals: approvals,
	}, nil
}

func randomScalar() (*ristretto255.Scalar, error) {
	uniform := make([]byte, 64)
	if _, err := rand.Read(uniform); err != nil {
		return nil, fmt.Errorf("read cryptographic randomness: %w", err)
	}
	return ristretto255.NewScalar().FromUniformBytes(uniform), nil
}

func schnorrChallenge(publicKey, commitment, message []byte) *ristretto255.Scalar {
	hash := sha512.New()
	for _, part := range [][]byte{[]byte("govdid:schnorr-approval:v1"), commitment, publicKey, message} {
		hash.Write(protocol.EncodeFields(part))
	}
	return ristretto255.NewScalar().FromUniformBytes(hash.Sum(nil))
}
