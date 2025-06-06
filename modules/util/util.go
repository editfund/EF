// Copyright 2017 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package util

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// IsEmptyString checks if the provided string is empty
func IsEmptyString(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

// NormalizeEOL will convert Windows (CRLF) and Mac (CR) EOLs to UNIX (LF)
func NormalizeEOL(input []byte) []byte {
	var right, left, pos int
	if right = bytes.IndexByte(input, '\r'); right == -1 {
		return input
	}
	length := len(input)
	tmp := make([]byte, length)

	// We know that left < length because otherwise right would be -1 from IndexByte.
	copy(tmp[pos:pos+right], input[left:left+right])
	pos += right
	tmp[pos] = '\n'
	left += right + 1
	pos++

	for left < length {
		if input[left] == '\n' {
			left++
		}

		right = bytes.IndexByte(input[left:], '\r')
		if right == -1 {
			copy(tmp[pos:], input[left:])
			pos += length - left
			break
		}
		copy(tmp[pos:pos+right], input[left:left+right])
		pos += right
		tmp[pos] = '\n'
		left += right + 1
		pos++
	}
	return tmp[:pos]
}

// CryptoRandomInt returns a crypto random integer between 0 and limit, inclusive
func CryptoRandomInt(limit int64) (int64, error) {
	rInt, err := rand.Int(rand.Reader, big.NewInt(limit))
	if err != nil {
		return 0, err
	}
	return rInt.Int64(), nil
}

const alphanumericalChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// CryptoRandomString generates a crypto random alphanumerical string, each byte is generated by [0,61] range
func CryptoRandomString(length int64) (string, error) {
	buf := make([]byte, length)
	limit := int64(len(alphanumericalChars))
	for i := range buf {
		num, err := CryptoRandomInt(limit)
		if err != nil {
			return "", err
		}
		buf[i] = alphanumericalChars[num]
	}
	return string(buf), nil
}

// CryptoRandomBytes generates `length` crypto bytes
// This differs from CryptoRandomString, as each byte in CryptoRandomString is generated by [0,61] range
// This function generates totally random bytes, each byte is generated by [0,255] range
func CryptoRandomBytes(length int64) []byte {
	// crypto/rand.Read is documented to never return a error.
	// https://go.dev/issue/66821
	buf := make([]byte, length)
	n, err := rand.Read(buf)
	if err != nil || n != int(length) {
		panic(err)
	}

	return buf
}

// ToUpperASCII returns s with all ASCII letters mapped to their upper case.
func ToUpperASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'a' <= c && c <= 'z' {
			b[i] -= 'a' - 'A'
		}
	}
	return string(b)
}

// ToTitleCase returns s with all english words capitalized
func ToTitleCase(s string) string {
	// `cases.Title` is not thread-safe, do not use global shared variable for it
	return cases.Title(language.English).String(s)
}

// ToTitleCaseNoLower returns s with all english words capitalized without lower-casing
func ToTitleCaseNoLower(s string) string {
	// `cases.Title` is not thread-safe, do not use global shared variable for it
	return cases.Title(language.English, cases.NoLower).String(s)
}

// ToInt64 transform a given int into int64.
func ToInt64(number any) (int64, error) {
	var value int64
	switch v := number.(type) {
	case int:
		value = int64(v)
	case int8:
		value = int64(v)
	case int16:
		value = int64(v)
	case int32:
		value = int64(v)
	case int64:
		value = v

	case uint:
		value = int64(v)
	case uint8:
		value = int64(v)
	case uint16:
		value = int64(v)
	case uint32:
		value = int64(v)
	case uint64:
		value = int64(v)

	case float32:
		value = int64(v)
	case float64:
		value = int64(v)

	case string:
		var err error
		if value, err = strconv.ParseInt(v, 10, 64); err != nil {
			return 0, err
		}
	default:
		return 0, fmt.Errorf("unable to convert %v to int64", number)
	}
	return value, nil
}

// ToFloat64 transform a given int into float64.
func ToFloat64(number any) (float64, error) {
	var value float64
	switch v := number.(type) {
	case int:
		value = float64(v)
	case int8:
		value = float64(v)
	case int16:
		value = float64(v)
	case int32:
		value = float64(v)
	case int64:
		value = float64(v)

	case uint:
		value = float64(v)
	case uint8:
		value = float64(v)
	case uint16:
		value = float64(v)
	case uint32:
		value = float64(v)
	case uint64:
		value = float64(v)

	case float32:
		value = float64(v)
	case float64:
		value = v

	case string:
		var err error
		if value, err = strconv.ParseFloat(v, 64); err != nil {
			return 0, err
		}
	default:
		return 0, fmt.Errorf("unable to convert %v to float64", number)
	}
	return value, nil
}

// ToPointer returns the pointer of a copy of any given value
func ToPointer[T any](val T) *T {
	return &val
}

// Iif is an "inline-if", it returns "trueVal" if "condition" is true, otherwise "falseVal"
func Iif[T any](condition bool, trueVal, falseVal T) T {
	if condition {
		return trueVal
	}
	return falseVal
}

// IfZero returns "def" if "v" is a zero value, otherwise "v"
func IfZero[T comparable](v, def T) T {
	var zero T
	if v == zero {
		return def
	}
	return v
}

// OptionalArg helps the "optional argument" in Golang:
//
//	func foo(optArg ...int) { return OptionalArg(optArg) }
//		calling `foo()` gets zero value 0, calling `foo(100)` gets 100
//	func bar(optArg ...int) { return OptionalArg(optArg, 42) }
//		calling `bar()` gets default value 42, calling `bar(100)` gets 100
//
// Passing more than 1 item to `optArg` or `defaultValue` is undefined behavior.
// At the moment only the first item is used.
func OptionalArg[T any](optArg []T, defaultValue ...T) (ret T) {
	if len(optArg) >= 1 {
		return optArg[0]
	}
	if len(defaultValue) >= 1 {
		return defaultValue[0]
	}
	return ret
}

func ReserveLineBreakForTextarea(input string) string {
	// Since the content is from a form which is a textarea, the line endings are \r\n.
	// It's a standard behavior of HTML.
	// But we want to store them as \n like what GitHub does.
	// And users are unlikely to really need to keep the \r.
	// Other than this, we should respect the original content, even leading or trailing spaces.
	return strings.ReplaceAll(input, "\r\n", "\n")
}

// GenerateSSHKeypair generates a ed25519 SSH-compatible keypair.
func GenerateSSHKeypair() (publicKey, privateKey []byte, err error) {
	public, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("ed25519.GenerateKey: %w", err)
	}

	privPEM, err := ssh.MarshalPrivateKey(private, "")
	if err != nil {
		return nil, nil, fmt.Errorf("ssh.MarshalPrivateKey: %w", err)
	}

	sshPublicKey, err := ssh.NewPublicKey(public)
	if err != nil {
		return nil, nil, fmt.Errorf("ssh.NewPublicKey: %w", err)
	}

	return ssh.MarshalAuthorizedKey(sshPublicKey), pem.EncodeToMemory(privPEM), nil
}
