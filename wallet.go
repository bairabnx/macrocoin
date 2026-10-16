package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"fmt"
)

func main() {
	// Generate an Elliptic Curve Cryptography (ECC) Private Key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Println("Error generating cryptographic keys:", err)
		return
	}

	// Extract the matching unique Public Key component
	publicKey := &privateKey.PublicKey

	// Encode keys into human-readable hex string values using the correct x509 library
	privBytes, _ := x509.MarshalECPrivateKey(privateKey)
	pubBytes, _ := x509.MarshalPKIXPublicKey(publicKey)

	fmt.Println("=================================================================")
	fmt.Println("🔐 NEW MACROCOIN SECURE WALLET CREATED")
	fmt.Println("=================================================================")
	fmt.Printf("🤫 PRIVATE KEY (Keep secret!):\n%x\n\n", privBytes)
	fmt.Printf("🌐 PUBLIC KEY  (Your wallet address):\n%x\n", pubBytes)
	fmt.Println("=================================================================")
}
