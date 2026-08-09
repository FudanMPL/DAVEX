package lsag

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"didcontract/protocol"
	"github.com/gtank/ristretto255"
)

const Algorithm = "DID-LSAG-Ristretto255-v1"

// ComputeKeyImage derives the contextual key image without creating a random
// LSAG signature. It uses the same link domain as Sign and the contract.
func ComputeKeyImage(privateKeyHex string, publicKeys []string, policyID, qID string, timeSlot int64) (string, error) {
	if len(publicKeys) == 0 || len(publicKeys) > 200 {
		return "", errors.New("LSAG ring size must be between 1 and 200")
	}
	secret, err := decodeScalar(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("signer private key: %w", err)
	}
	signerPublic := ristretto255.NewElement().ScalarBaseMult(secret)
	occurrences := 0
	for index, encoded := range publicKeys {
		point, err := decodePoint(encoded)
		if err != nil {
			return "", fmt.Errorf("ring member %d: %w", index, err)
		}
		if point.Equal(signerPublic) == 1 {
			occurrences++
		}
	}
	if occurrences != 1 {
		return "", errors.New("signer public key must occur exactly once in ring")
	}
	hctx, err := hashToPoint(protocol.LinkDomainBytes(publicKeys, policyID, qID, timeSlot))
	if err != nil {
		return "", err
	}
	return encodePoint(ristretto255.NewElement().ScalarMult(secret, hctx)), nil
}

func Sign(privateKeyHex string, publicKeys []string, context protocol.MessageContext) (protocol.LSAGSignature, error) {
	if len(publicKeys) == 0 || len(publicKeys) > 200 {
		return protocol.LSAGSignature{}, errors.New("LSAG ring size must be between 1 and 200")
	}
	secret, err := decodeScalar(privateKeyHex)
	if err != nil {
		return protocol.LSAGSignature{}, fmt.Errorf("signer private key: %w", err)
	}
	points := make([]*ristretto255.Element, len(publicKeys))
	encodedPoints := make([][]byte, len(publicKeys))
	signerPublic := ristretto255.NewElement().ScalarBaseMult(secret)
	signer := -1
	for i, encoded := range publicKeys {
		points[i], err = decodePoint(encoded)
		if err != nil {
			return protocol.LSAGSignature{}, fmt.Errorf("ring member %d: %w", i, err)
		}
		encodedPoints[i] = points[i].Encode(nil)
		if points[i].Equal(signerPublic) == 1 {
			if signer >= 0 {
				return protocol.LSAGSignature{}, errors.New("signer public key occurs more than once in ring")
			}
			signer = i
		}
	}
	if signer < 0 {
		return protocol.LSAGSignature{}, errors.New("signer public key is absent from ring")
	}
	hctx, err := hashToPoint(protocol.LinkDomainBytes(publicKeys, context.PolicyID, context.QID, context.TimeSlot))
	if err != nil {
		return protocol.LSAGSignature{}, err
	}
	keyImage := ristretto255.NewElement().ScalarMult(secret, hctx)
	ringEncoded := protocol.EncodeFields(encodedPoints...)
	message := protocol.MessageContextBytes(context)
	responses := make([]*ristretto255.Scalar, len(publicKeys))
	challenges := make([]*ristretto255.Scalar, len(publicKeys))
	alpha, err := randomScalar()
	if err != nil {
		return protocol.LSAGSignature{}, err
	}
	left := ristretto255.NewElement().ScalarBaseMult(alpha)
	right := ristretto255.NewElement().ScalarMult(alpha, hctx)
	next := (signer + 1) % len(publicKeys)
	challenges[next] = challenge(ringEncoded, keyImage.Encode(nil), message, left.Encode(nil), right.Encode(nil))
	for i := next; i != signer; i = (i + 1) % len(publicKeys) {
		responses[i], err = randomScalar()
		if err != nil {
			return protocol.LSAGSignature{}, err
		}
		rG := ristretto255.NewElement().ScalarBaseMult(responses[i])
		cP := ristretto255.NewElement().ScalarMult(challenges[i], points[i])
		left = ristretto255.NewElement().Add(rG, cP)
		rH := ristretto255.NewElement().ScalarMult(responses[i], hctx)
		cI := ristretto255.NewElement().ScalarMult(challenges[i], keyImage)
		right = ristretto255.NewElement().Add(rH, cI)
		challenges[(i+1)%len(publicKeys)] = challenge(ringEncoded, keyImage.Encode(nil), message, left.Encode(nil), right.Encode(nil))
	}
	product := ristretto255.NewScalar().Multiply(challenges[signer], secret)
	responses[signer] = ristretto255.NewScalar().Subtract(alpha, product)
	encodedResponses := make([]string, len(responses))
	for i, response := range responses {
		encodedResponses[i] = encodeScalar(response)
	}
	return protocol.LSAGSignature{Algorithm: Algorithm, C0: encodeScalar(challenges[0]), Responses: encodedResponses, KeyImage: encodePoint(keyImage)}, nil
}

func Verify(publicKeys []string, context protocol.MessageContext, signature protocol.LSAGSignature) error {
	if signature.Algorithm != Algorithm || len(publicKeys) == 0 || len(signature.Responses) != len(publicKeys) {
		return errors.New("invalid LSAG algorithm, ring, or response size")
	}
	points := make([]*ristretto255.Element, len(publicKeys))
	encodedPoints := make([][]byte, len(publicKeys))
	var err error
	for i, encoded := range publicKeys {
		points[i], err = decodePoint(encoded)
		if err != nil {
			return err
		}
		encodedPoints[i] = points[i].Encode(nil)
	}
	keyImage, err := decodePoint(signature.KeyImage)
	if err != nil {
		return err
	}
	c0, err := decodeScalar(signature.C0)
	if err != nil {
		return err
	}
	hctx, err := hashToPoint(protocol.LinkDomainBytes(publicKeys, context.PolicyID, context.QID, context.TimeSlot))
	if err != nil {
		return err
	}
	ringEncoded := protocol.EncodeFields(encodedPoints...)
	message := protocol.MessageContextBytes(context)
	c := c0
	for i, encodedResponse := range signature.Responses {
		response, err := decodeScalar(encodedResponse)
		if err != nil {
			return err
		}
		left := ristretto255.NewElement().Add(
			ristretto255.NewElement().ScalarBaseMult(response),
			ristretto255.NewElement().ScalarMult(c, points[i]),
		)
		right := ristretto255.NewElement().Add(
			ristretto255.NewElement().ScalarMult(response, hctx),
			ristretto255.NewElement().ScalarMult(c, keyImage),
		)
		c = challenge(ringEncoded, keyImage.Encode(nil), message, left.Encode(nil), right.Encode(nil))
	}
	if c.Equal(c0) != 1 {
		return errors.New("LSAG challenge chain does not close")
	}
	return nil
}

func randomScalar() (*ristretto255.Scalar, error) {
	uniform := make([]byte, 64)
	if _, err := rand.Read(uniform); err != nil {
		return nil, fmt.Errorf("read cryptographic randomness: %w", err)
	}
	return ristretto255.NewScalar().FromUniformBytes(uniform), nil
}

func hashToScalar(parts ...[]byte) *ristretto255.Scalar {
	hash := sha512.New()
	for _, part := range parts {
		hash.Write(protocol.EncodeFields(part))
	}
	return ristretto255.NewScalar().FromUniformBytes(hash.Sum(nil))
}

func hashToPoint(domain []byte) (*ristretto255.Element, error) {
	sum := sha512.Sum512(protocol.EncodeFields([]byte("govdid:hash-to-point:v1"), domain))
	point := ristretto255.NewElement().FromUniformBytes(sum[:])
	if point.Equal(ristretto255.NewElement().Zero()) == 1 {
		return nil, errors.New("hash-to-point returned identity")
	}
	return point, nil
}

func challenge(ring, keyImage, message, left, right []byte) *ristretto255.Scalar {
	return hashToScalar([]byte("govdid:lsag-challenge:v1"), ring, keyImage, message, left, right)
}

func decodePoint(value string) (*ristretto255.Element, error) {
	b, err := hex.DecodeString(stripHex(value))
	if err != nil || len(b) != 32 {
		return nil, errors.New("invalid Ristretto255 point encoding")
	}
	point := ristretto255.NewElement()
	if err := point.Decode(b); err != nil || point.Equal(ristretto255.NewElement().Zero()) == 1 {
		return nil, errors.New("invalid Ristretto255 point")
	}
	return point, nil
}

func decodeScalar(value string) (*ristretto255.Scalar, error) {
	b, err := hex.DecodeString(stripHex(value))
	if err != nil || len(b) != 32 {
		return nil, errors.New("invalid Ristretto255 scalar encoding")
	}
	scalar := ristretto255.NewScalar()
	if err := scalar.Decode(b); err != nil {
		return nil, errors.New("non-canonical Ristretto255 scalar")
	}
	return scalar, nil
}

func stripHex(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "0x")
	return strings.TrimPrefix(value, "0X")
}

func encodePoint(point *ristretto255.Element) string  { return hex.EncodeToString(point.Encode(nil)) }
func encodeScalar(scalar *ristretto255.Scalar) string { return hex.EncodeToString(scalar.Encode(nil)) }
