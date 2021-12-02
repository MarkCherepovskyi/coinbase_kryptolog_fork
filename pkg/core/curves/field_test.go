//
// Copyright Coinbase, Inc. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0
//

package curves

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	one                = big.NewInt(1)
	modulus, modulusOk = new(big.Int).SetString(
		"1000000000000000000000000000000014DEF9DEA2F79CD65812631A5CF5D3ED",
		16,
	)
	oneBelowModulus = zero().Sub(modulus, one)
	oneAboveModulus = zero().Add(modulus, one)
	field25519      = NewField(modulus)
)

type buggedReader struct{}

func (r buggedReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("EOF")
}

func zero() *big.Int {
	return new(big.Int)
}

func assertElementZero(t *testing.T, e *Element) {
	assert.Equal(t, zero().Bytes(), e.Bytes())
}

type binaryOperation func(*Element) *Element

func assertUnequalFieldsPanic(t *testing.T, b binaryOperation) {
	altField := NewField(big.NewInt(23))
	altElement := altField.NewElement(one)

	assert.PanicsWithValue(
		t,
		"fields must match for valid binary operation",
		func() { b(altElement) },
	)
}

func TestFieldModulus(t *testing.T) {
	assert.True(t, modulusOk)
}

func TestNewField(t *testing.T) {
	assert.PanicsWithValue(
		t,
		fmt.Sprintf("modulus: %x is not a prime", oneBelowModulus),
		func() { NewField(oneBelowModulus) },
	)
	assert.NotPanics(
		t,
		func() { NewField(modulus) },
	)
}

func TestNewElement(t *testing.T) {
	assert.PanicsWithValue(
		t,
		fmt.Sprintf("value: %x is not within field: %x", modulus, field25519.Int),
		func() { newElement(field25519, modulus) },
	)
	assert.NotPanics(
		t,
		func() { newElement(field25519, oneBelowModulus) },
	)
}

func TestElementIsValid(t *testing.T) {
	assert.False(t, field25519.IsValid(zero().Neg(one)))
	assert.False(t, field25519.IsValid(modulus))
	assert.False(t, field25519.IsValid(oneAboveModulus))
	assert.True(t, field25519.IsValid(oneBelowModulus))
}

func TestFieldNewElement(t *testing.T) {
	element := field25519.NewElement(oneBelowModulus)

	assert.Equal(t, oneBelowModulus, element.Value)
	assert.Equal(t, field25519, element.Field())
}

func TestZeroElement(t *testing.T) {
	assert.Equal(t, zero(), field25519.Zero().Value)
	assert.Equal(t, field25519, field25519.Zero().Field())
}

func TestOneElement(t *testing.T) {
	assert.Equal(t, field25519.One().Value, one)
	assert.Equal(t, field25519.One().Field(), field25519)
}

func TestRandomElement(t *testing.T) {
	randomElement1, err := field25519.RandomElement(nil)
	require.NoError(t, err)
	randomElement2, err := field25519.RandomElement(nil)
	require.NoError(t, err)
	randomElement3, err := field25519.RandomElement(new(buggedReader))
	require.Error(t, err)

	assert.Equal(t, field25519, randomElement1.Field())
	assert.Equal(t, field25519, randomElement2.Field())
	assert.NotEqual(t, randomElement1.Value, randomElement2.Value)
	assert.Nil(t, randomElement3)
}

func TestElementFromBytes(t *testing.T) {
	element := field25519.ElementFromBytes(oneBelowModulus.Bytes())

	assert.Equal(t, field25519, element.Field())
	assert.Equal(t, oneBelowModulus, element.Value)
}

func TestReducedElementFromBytes(t *testing.T) {
	element := field25519.ReducedElementFromBytes(oneBelowModulus.Bytes())

	assert.Equal(t, field25519, element.Field())
	assert.Equal(t, oneBelowModulus, element.Value)

	element = field25519.ReducedElementFromBytes(oneAboveModulus.Bytes())

	assert.Equal(t, field25519, element.Field())
	assert.Equal(t, one, element.Value)
}

func TestAddElement(t *testing.T) {
	element1 := field25519.NewElement(one)
	element2 := field25519.NewElement(big.NewInt(2))
	element3 := field25519.NewElement(oneBelowModulus)
	element4 := &Element{field25519, modulus}

	assert.Equal(t, element2, element1.Add(element1))
	assert.Equal(t, big.NewInt(3), element1.Add(element2).Value)
	assert.Equal(t, big.NewInt(3), element2.Add(element1).Value)
	assert.Equal(t, one, element1.Add(element4).Value)
	assert.Equal(t, one, element3.Add(element2).Value)
	assertElementZero(t, element1.Add(element3))
	assertUnequalFieldsPanic(t, element1.Add)
}

func TestSubElement(t *testing.T) {
	element1 := field25519.NewElement(one)
	element2 := field25519.NewElement(big.NewInt(2))
	element3 := field25519.NewElement(oneBelowModulus)
	element4 := &Element{field25519, modulus}

	assertElementZero(t, element1.Sub(element1))
	assert.Equal(t, element3, element1.Sub(element2))
	assert.Equal(t, element1, element2.Sub(element1))
	assert.Equal(t, element1, element1.Sub(element4))
	assert.Equal(t, element3, element4.Sub(element1))
	assert.Equal(t, element1, element4.Sub(element3))
	assert.Equal(t, element3, element3.Sub(element4))
	assertUnequalFieldsPanic(t, element1.Sub)
}

func TestMulElement(t *testing.T) {
	element1 := field25519.NewElement(one)
	element2 := field25519.NewElement(big.NewInt(2))
	element3 := field25519.NewElement(oneBelowModulus)
	element4 := field25519.NewElement(zero())
	expectedProduct, ok := new(big.Int).SetString(
		"1000000000000000000000000000000014def9dea2f79cd65812631a5cf5d3eb",
		16,
	)
	require.True(t, ok)

	assertElementZero(t, element1.Mul(element4))
	assertElementZero(t, element4.Mul(element1))
	assert.Equal(t, element3, element1.Mul(element3))
	assert.Equal(t, element3, element3.Mul(element1))
	assert.Equal(t, expectedProduct, element3.Mul(element2).Value)
	assert.Equal(t, expectedProduct, element2.Mul(element3).Value)
	assertUnequalFieldsPanic(t, element1.Mul)
}

func TestDivElement(t *testing.T) {
	element1 := field25519.NewElement(one)
	element2 := field25519.NewElement(big.NewInt(2))
	element3 := field25519.NewElement(oneBelowModulus)
	element4 := field25519.NewElement(zero())
	expectedQuotient1, ok := new(big.Int).SetString(
		"80000000000000000000000000000000a6f7cef517bce6b2c09318d2e7ae9f6",
		16,
	)
	require.True(t, ok)
	expectedQuotient2, ok := new(big.Int).SetString(
		"1000000000000000000000000000000014def9dea2f79cd65812631a5cf5d3eb",
		16,
	)
	require.True(t, ok)

	assertElementZero(t, element4.Div(element3))
	assert.Equal(t, element3, element3.Div(element1))
	assert.Equal(t, expectedQuotient1, element3.Div(element2).Value)
	assert.Equal(t, expectedQuotient2, element2.Div(element3).Value)
	assert.Panics(t, func() { element3.Div(element4) })
	assertUnequalFieldsPanic(t, element1.Div)
}

func TestIsEqualElement(t *testing.T) {
	element1 := field25519.NewElement(oneBelowModulus)
	element2 := field25519.NewElement(big.NewInt(23))
	element3 := field25519.NewElement(oneBelowModulus)
	altField := NewField(big.NewInt(23))
	altElement1 := altField.NewElement(one)

	assert.False(t, element1.IsEqual(element2))
	assert.True(t, element1.IsEqual(element3))
	assert.True(t, element1.IsEqual(element1))
	assert.False(t, element1.IsEqual(altElement1))
}

func TestBigIntElement(t *testing.T) {
	element := field25519.NewElement(oneBelowModulus)

	assert.Equal(t, oneBelowModulus, element.BigInt())
}

func TestBytesElement(t *testing.T) {
	element := field25519.NewElement(oneBelowModulus)

	assert.Equal(
		t,
		[]byte{
			0x10, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x14, 0xde, 0xf9, 0xde, 0xa2,
			0xf7, 0x9c, 0xd6, 0x58, 0x12, 0x63, 0x1a, 0x5c, 0xf5,
			0xd3, 0xec,
		},
		element.Bytes(),
	)
}

func TestCloneElement(t *testing.T) {
	element := field25519.NewElement(oneBelowModulus)
	clone := element.Clone()

	assert.Equal(t, clone, element)

	clone.Value.Add(one, one)

	assert.NotEqual(t, clone, element)
}

// Tests un/marshaling Element
func TestElementMarshalJsonRoundTrip(t *testing.T) {
	// TODO: test really large numbers for field25519/value
	ins := []*Element{
		newElement(field25519, big.NewInt(300)),
		newElement(field25519, big.NewInt(300000)),
		newElement(field25519, big.NewInt(12812798)),
		newElement(field25519, big.NewInt(17)),
		newElement(field25519, big.NewInt(5066680)),
		newElement(field25519, big.NewInt(3005)),
		newElement(field25519, big.NewInt(317)),
		newElement(field25519, big.NewInt(323)),
		newElement(field25519, oneBelowModulus),
	}

	// Run all the tests!
	for _, in := range ins {
		bytes, err := json.Marshal(in)
		require.NoError(t, err)
		require.NotNil(t, bytes)

		// Unmarshal and test
		out := &Element{}
		err = json.Unmarshal(bytes, &out)
		require.NoError(t, err)
		require.NotNil(t, out)
		require.NotNil(t, out.Modulus)
		require.NotNil(t, out.Value)

		assert.Equal(t, in.Modulus.Bytes(), out.Modulus.Bytes())
		assert.Equal(t, in.Value.Bytes(), out.Value.Bytes())
	}
}
