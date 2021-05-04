package crypto

import (
	. "crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"fmt"
)

// bytes to private key
func BytesToPrivateKey(priv []byte) *PrivateKey {
	block, _ := pem.Decode(priv)
	enc := x509.IsEncryptedPEMBlock(block)
	b := block.Bytes
	// var err error
	if enc {
		fmt.Println("is encrypted pem block")
		b, _ = x509.DecryptPEMBlock(block, nil)
	}
	key, _ := x509.ParsePKCS1PrivateKey(b)
	return key
}

// bytes to public key
func BytesToPublicKey(pub []byte) *PublicKey {
	block, _ := pem.Decode(pub)
	enc := x509.IsEncryptedPEMBlock(block)
	b := block.Bytes
	if enc {
		fmt.Println("is encrypted pem block")
		b, _ = x509.DecryptPEMBlock(block, nil)
	}
	ifc, _ := x509.ParsePKIXPublicKey(b)
	key, ok := ifc.(*PublicKey)
	if !ok {
		fmt.Println("not ok")
	}
	return key
}
