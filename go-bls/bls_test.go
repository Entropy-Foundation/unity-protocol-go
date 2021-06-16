/**
 * File        : bls_test.go
 * Description : Unit tests.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Stable
 *
 * This module provides unit tests for the Boneh-Lynn-Shacham signature scheme.
 */

package bls

import (
	"crypto/sha256"
	"fmt"
	. "github.com/unity-go/go-bls"
	"math/rand"
	"testing"
	"time"
)

func TestSignVerify(test *testing.T) {

	message := "This is a message."

	// Generate a key pair.
	params := GenParamsTypeA(160, 512)
	pairing := GenPairing(params)
	system, err := GenSystem(pairing)
	if err != nil {
		test.Fatal(err)
	}
	key, secret, err := GenKeys(system)
	if err != nil {
		test.Fatal(err)
	}

	// Sign the message.
	hash := sha256.Sum256([]byte(message))
	signature := Sign(hash, secret)

	// Verify the signature.
	if !Verify(signature, hash, key) {
		test.Fatal("Failed to verify signature.")
	}

	// Clean up.
	signature.Free()
	key.Free()
	secret.Free()
	system.Free()
	pairing.Free()
	params.Free()

}

func TestAggregateVerify(test *testing.T) {

	messages := []string{
		"This is a message.",
		"This is another message.",
		"This is yet another message.",
		"These messages are unique.",
	}
	n := len(messages)

	// Generate key pairs.
	params, err := GenParamsTypeD(9563, 512)
	if err != nil {
		test.Fatal(err)
	}
	pairing := GenPairing(params)
	system, err := GenSystem(pairing)
	if err != nil {
		test.Fatal(err)
	}
	keys := make([]PublicKey, n)
	secrets := make([]PrivateKey, n)
	for i := 0; i < n; i++ {
		keys[i], secrets[i], err = GenKeys(system)
		if err != nil {
			test.Fatal(err)
		}
	}

	// Sign the messages.
	hashes := make([][sha256.Size]byte, n)
	signatures := make([]Signature, n)
	for i := 0; i < n; i++ {
		hashes[i] = sha256.Sum256([]byte(messages[i]))
		signatures[i] = Sign(hashes[i], secrets[i])
	}

	// Aggregate the signatures.
	aggregate, err := Aggregate(signatures, system)
	if err != nil {
		test.Fatal(err)
	}

	// Verify the aggregate signature.
	valid, err := AggregateVerify(aggregate, hashes, keys)
	if err != nil {
		test.Fatal(err)
	}
	if !valid {
		test.Fatal("Failed to verify aggregate signature.")
	}

	// Clean up.
	aggregate.Free()
	for i := 0; i < n; i++ {
		signatures[i].Free()
		keys[i].Free()
		secrets[i].Free()
	}
	system.Free()
	pairing.Free()
	params.Free()

}

func TestThresholdSignature(test *testing.T) {

	message := "This is a message."

	// Generate key shares.
	params := GenParamsTypeF(32)
	pairing := GenPairing(params)
	system, _ := GenSystem(pairing)

	rand.Seed(time.Now().UnixNano())
	n := 7
	t := 5

	groupKey, memberKeys, groupSecret, memberSecrets, _ := GenKeyShares(t, n, system)

	paras, _ := params.ToBytes()
	fmt.Println(paras, "1")
	paramss, _ := ParamsFromBytes(paras)
	pairings := GenPairing(paramss)
	sysToByte := system.ToBytes()
	s, _ := SystemFromBytes(pairings, sysToByte)

	// Select group members.
	memberIds := rand.Perm(n)[:t]

	// Sign the message.
	hash := sha256.Sum256([]byte(message))
	shares := make([]Signature, t)

	for i := 0; i < t; i++ {
		inBy := system.PrivateKeyToBytes(memberSecrets[memberIds[i]])
		ori := s.PrivateFromBytes(inBy)
		shares[i] = Sign(hash, ori)
	}
	fmt.Println(shares)

	// Recover the threshold signature.
	signature, _ := Threshold(shares, memberIds, s)

	inBy := system.PublicKeyToBytes(groupKey)
	ori, _ := system.PublicFromBytes(inBy)
	// Verify the threshold signature.

	// inBy2 := system.PrivateKeyToBytes(groupSecret)
	// ori, _ := system.PrivateFromBytes2(inBy2)

	// hash2 := sha256.Sum256([]byte("hi"))
	start6 := time.Now()

	if !Verify(signature, hash, ori) {
		test.Fatal("Failed to verify signature.")

	}
	fmt.Println("*********************************", time.Since(start6))

	// Clean up.
	signature.Free()
	groupKey.Free()
	groupSecret.Free()
	for i := 0; i < t; i++ {
		shares[i].Free()
	}
	for i := 0; i < n; i++ {
		memberKeys[i].Free()
		memberSecrets[i].Free()
	}
	system.Free()
	pairing.Free()
	params.Free()
}

func TestToFromBytes(test *testing.T) {

	message := "This is a message."

	// Generate a key pair.
	params := GenParamsTypeA(160, 512)
	pairing := GenPairing(params)
	system, err := GenSystem(pairing)
	if err != nil {
		test.Fatal(err)
	}
	key, secret, err := GenKeys(system)
	if err != nil {
		test.Fatal(err)
	}

	// Sign the message and serialize the signature.
	hash := sha256.Sum256([]byte(message))
	signatureOut := Sign(hash, secret)
	bytes := system.SigToBytes(signatureOut)

	// Deserialize the signature and verify it.
	signatureIn, err := system.SigFromBytes(bytes)
	if err != nil {
		test.Fatal(err)
	}
	if !Verify(signatureIn, hash, key) {
		test.Fatal("Failed to verify signature.")
	}

	// Clean up.
	signatureIn.Free()
	signatureOut.Free()
	key.Free()
	secret.Free()
	system.Free()
	pairing.Free()
	params.Free()

}

func BenchmarkVerify(benchmark *testing.B) {

	message := "This is a message."

	// Generate a key pair.
	params := GenParamsTypeF(160)
	pairing := GenPairing(params)
	system, err := GenSystem(pairing)
	if err != nil {
		benchmark.Fatal(err)
	}
	key, secret, err := GenKeys(system)
	if err != nil {
		benchmark.Fatal(err)
	}

	// Sign the message.
	hash := sha256.Sum256([]byte(message))
	signature := Sign(hash, secret)

	// Verify the signature.
	benchmark.StartTimer()
	for i := 0; i < benchmark.N; i++ {
		Verify(signature, hash, key)
	}
	benchmark.StopTimer()

	// Clean up.
	signature.Free()
	key.Free()
	secret.Free()
	system.Free()
	pairing.Free()
	params.Free()

}
