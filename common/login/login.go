package login

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/hex"

	"github.com/rotisserie/eris"
)

const (
	decryptionErrorMsg = "cannot decrypt access token"
)

type Encryption struct {
	curve      ecdh.Curve
	privateKey *ecdh.PrivateKey
	publicKey  *ecdh.PublicKey
}

type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	PublicKey   string `json:"pub_key"`
	Nonce       string `json:"nonce"`
}

// NewEncryption creates a new Encryption struct.
func NewEncryption() (Encryption, error) {
	enc := Encryption{}
	err := enc.generateKeys()
	if err != nil {
		return enc, err
	}
	return enc, nil
}

// generateKeys generates a private and public key pair.
func (enc *Encryption) generateKeys() error {
	enc.curve = ecdh.P256()
	privateKey, err := enc.curve.GenerateKey(rand.Reader)
	if err != nil {
		return eris.Wrap(err, "cannot generate keys")
	}
	enc.privateKey = privateKey
	enc.publicKey = privateKey.PublicKey()
	return nil
}

// encodedPublicKey returns the public key as a hex string.
func (enc Encryption) EncodedPublicKey() string {
	return hex.EncodeToString(enc.publicKey.Bytes())
}

// decryptAccessToken decrypts the access token using the private key and nonce.
func (enc Encryption) DecryptAccessToken(accessToken string, publicKey string, nonce string) (string, error) {
	decodedAccessToken, err := hex.DecodeString(accessToken)
	if err != nil {
		return "", eris.Wrap(err, decryptionErrorMsg)
	}

	decodedNonce, err := hex.DecodeString(nonce)
	if err != nil {
		return "", eris.Wrap(err, decryptionErrorMsg)
	}

	decodedPublicKey, err := hex.DecodeString(publicKey)
	if err != nil {
		return "", eris.Wrap(err, decryptionErrorMsg)
	}

	remotePublicKey, err := enc.curve.NewPublicKey(decodedPublicKey)
	if err != nil {
		return "", eris.Wrap(err, decryptionErrorMsg)
	}

	secret, err := enc.privateKey.ECDH(remotePublicKey)
	if err != nil {
		return "", eris.Wrap(err, decryptionErrorMsg)
	}

	block, err := aes.NewCipher(secret)
	if err != nil {
		return "", eris.Wrap(err, decryptionErrorMsg)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", eris.Wrap(err, decryptionErrorMsg)
	}

	decryptedAccessToken, err := aesgcm.Open(nil, decodedNonce, decodedAccessToken, nil)
	if err != nil {
		return "", eris.Wrap(err, decryptionErrorMsg)
	}

	return string(decryptedAccessToken), nil
}
