package login

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecryptAccessToken(t *testing.T) {
	// Initialize encryption
	enc, err := NewEncryption()
	require.NoError(t, err)

	// Generate a remote key pair
	remoteCurve := ecdh.P256()
	remotePrivateKey, err := remoteCurve.GenerateKey(rand.Reader)
	require.NoError(t, err)
	remotePublicKey := remotePrivateKey.PublicKey()

	// Sample access token and nonce
	accessToken := "sample_access_token"
	nonce := make([]byte, 12)
	_, err = rand.Read(nonce)
	require.NoError(t, err)

	// Encrypt the access token
	secret, err := remotePrivateKey.ECDH(enc.publicKey)
	require.NoError(t, err)

	block, err := aes.NewCipher(secret)
	require.NoError(t, err)

	aesgcm, err := cipher.NewGCM(block)
	require.NoError(t, err)

	ciphertext := aesgcm.Seal(nil, nonce, []byte(accessToken), nil)

	// Encode values to hex strings
	hexAccessToken := hex.EncodeToString(ciphertext)
	hexNonce := hex.EncodeToString(nonce)
	hexPublicKey := hex.EncodeToString(remotePublicKey.Bytes())

	// Call the function
	decryptedAccessToken, err := enc.DecryptAccessToken(hexAccessToken, hexPublicKey, hexNonce)

	// Assertions
	require.NoError(t, err)
	assert.Equal(t, accessToken, decryptedAccessToken)
}
