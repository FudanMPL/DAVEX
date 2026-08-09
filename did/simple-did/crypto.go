package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"didcontract/protocol"
	"github.com/gtank/ristretto255"
)

func stripHexPrefix(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "0x")
	return strings.TrimPrefix(value, "0X")
}

func normalizeHex(value string) string {
	return strings.ToLower(stripHexPrefix(value))
}

func verifyECDSAP256(publicKeyHex, signatureHex string, message []byte) error {
	publicKeyBytes, err := hex.DecodeString(stripHexPrefix(publicKeyHex))
	if err != nil {
		return errors.New("invalid P-256 public key hex")
	}
	x, y := elliptic.Unmarshal(elliptic.P256(), publicKeyBytes)
	if x == nil || !elliptic.P256().IsOnCurve(x, y) {
		return errors.New("invalid P-256 public key point")
	}
	signature, err := hex.DecodeString(stripHexPrefix(signatureHex))
	if err != nil {
		return errors.New("invalid ECDSA signature hex")
	}
	digest := sha256.Sum256(message)
	if !ecdsa.VerifyASN1(&ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}, digest[:], signature) {
		return errors.New("ECDSA signature verification failed")
	}
	return nil
}

func validateP256PublicKey(publicKeyHex string) error {
	data, err := hex.DecodeString(stripHexPrefix(publicKeyHex))
	if err != nil {
		return errors.New("invalid P-256 public key hex")
	}
	x, y := elliptic.Unmarshal(elliptic.P256(), data)
	if x == nil || !elliptic.P256().IsOnCurve(x, y) {
		return errors.New("invalid P-256 public key point")
	}
	return nil
}

func validateLSAGPoint(pointHex string) error {
	b, err := hex.DecodeString(stripHexPrefix(pointHex))
	if err != nil || len(b) != 32 {
		return errors.New("invalid Ristretto255 point encoding")
	}
	p := ristretto255.NewElement()
	if err := p.Decode(b); err != nil {
		return errors.New("invalid Ristretto255 point")
	}
	if p.Equal(ristretto255.NewElement().Zero()) == 1 {
		return errors.New("identity Ristretto255 point is forbidden")
	}
	return nil
}

func decodeLSAGPoint(value string) (*ristretto255.Element, error) {
	if err := validateLSAGPoint(value); err != nil {
		return nil, err
	}
	b, _ := hex.DecodeString(stripHexPrefix(value))
	p := ristretto255.NewElement()
	if err := p.Decode(b); err != nil {
		return nil, err
	}
	return p, nil
}

func decodeLSAGScalar(value string) (*ristretto255.Scalar, error) {
	b, err := hex.DecodeString(stripHexPrefix(value))
	if err != nil || len(b) != 32 {
		return nil, errors.New("invalid Ristretto255 scalar encoding")
	}
	s := ristretto255.NewScalar()
	if err := s.Decode(b); err != nil {
		return nil, errors.New("non-canonical Ristretto255 scalar")
	}
	return s, nil
}

func hashToScalar(parts ...[]byte) *ristretto255.Scalar {
	h := sha512.New()
	for _, part := range parts {
		h.Write(protocol.EncodeFields(part))
	}
	uniform := h.Sum(nil)
	return ristretto255.NewScalar().FromUniformBytes(uniform)
}

func hashToPoint(domain []byte) (*ristretto255.Element, error) {
	sum := sha512.Sum512(protocol.EncodeFields([]byte("govdid:hash-to-point:v1"), domain))
	p := ristretto255.NewElement().FromUniformBytes(sum[:])
	if p.Equal(ristretto255.NewElement().Zero()) == 1 {
		return nil, errors.New("hash-to-point returned identity")
	}
	return p, nil
}

func lsagChallenge(ringEncoded, keyImageEncoded, message, left, right []byte) *ristretto255.Scalar {
	return hashToScalar([]byte("govdid:lsag-challenge:v1"), ringEncoded, keyImageEncoded, message, left, right)
}

func verifyLSAG(publicKeys []string, context protocol.MessageContext, signature protocol.LSAGSignature) error {
	if signature.Algorithm != "DID-LSAG-Ristretto255-v1" {
		return errors.New("unsupported LSAG algorithm")
	}
	if len(publicKeys) == 0 || len(publicKeys) > maxRingSize || len(signature.Responses) != len(publicKeys) {
		return errors.New("invalid LSAG ring or response size")
	}
	ring := make([]*ristretto255.Element, len(publicKeys))
	encodedKeys := make([][]byte, len(publicKeys))
	for i, publicKey := range publicKeys {
		point, err := decodeLSAGPoint(publicKey)
		if err != nil {
			return fmt.Errorf("ring member %d: %w", i, err)
		}
		ring[i] = point
		encodedKeys[i] = point.Encode(nil)
	}
	keyImage, err := decodeLSAGPoint(signature.KeyImage)
	if err != nil {
		return fmt.Errorf("key image: %w", err)
	}
	c0, err := decodeLSAGScalar(signature.C0)
	if err != nil {
		return fmt.Errorf("initial challenge: %w", err)
	}
	responses := make([]*ristretto255.Scalar, len(signature.Responses))
	for i, encoded := range signature.Responses {
		responses[i], err = decodeLSAGScalar(encoded)
		if err != nil {
			return fmt.Errorf("response %d: %w", i, err)
		}
	}
	ringEncoded := protocol.EncodeFields(encodedKeys...)
	linkDomain := protocol.LinkDomainBytes(publicKeys, context.PolicyID, context.QID, context.TimeSlot)
	hctx, err := hashToPoint(linkDomain)
	if err != nil {
		return err
	}
	message := protocol.MessageContextBytes(context)
	c := c0
	for i := range ring {
		rG := ristretto255.NewElement().ScalarBaseMult(responses[i])
		cP := ristretto255.NewElement().ScalarMult(c, ring[i])
		left := ristretto255.NewElement().Add(rG, cP)
		rH := ristretto255.NewElement().ScalarMult(responses[i], hctx)
		cI := ristretto255.NewElement().ScalarMult(c, keyImage)
		right := ristretto255.NewElement().Add(rH, cI)
		c = lsagChallenge(ringEncoded, keyImage.Encode(nil), message, left.Encode(nil), right.Encode(nil))
	}
	if c.Equal(c0) != 1 {
		return errors.New("LSAG challenge chain does not close")
	}
	return nil
}

func schnorrChallenge(publicKey, commitment, message []byte) *ristretto255.Scalar {
	return hashToScalar([]byte("govdid:schnorr-approval:v1"), commitment, publicKey, message)
}

func verifySchnorrRistretto(publicKeyHex, rHex, sHex string, message []byte) error {
	publicKey, err := decodeLSAGPoint(publicKeyHex)
	if err != nil {
		return fmt.Errorf("Schnorr public key: %w", err)
	}
	commitment, err := decodeLSAGPoint(rHex)
	if err != nil {
		return fmt.Errorf("Schnorr commitment: %w", err)
	}
	response, err := decodeLSAGScalar(sHex)
	if err != nil {
		return fmt.Errorf("Schnorr response: %w", err)
	}
	publicKeyBytes := publicKey.Encode(nil)
	commitmentBytes := commitment.Encode(nil)
	challenge := schnorrChallenge(publicKeyBytes, commitmentBytes, message)
	left := ristretto255.NewElement().ScalarBaseMult(response)
	eP := ristretto255.NewElement().ScalarMult(challenge, publicKey)
	right := ristretto255.NewElement().Add(commitment, eP)
	if left.Equal(right) != 1 {
		return errors.New("Schnorr signature verification failed")
	}
	return nil
}
