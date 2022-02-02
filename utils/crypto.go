/*
@Time : 2021/4/22 6:14 下午
@Author : shiguantian
@File : crypto
@Software: GoLand
*/
package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
)

const (
	Md5 = iota
	SHA1
	SHA256
	SHA512
)

func HashSign(hashType int, text string, base64Encode bool) (string, error) {
	var hash hash.Hash
	switch hashType {
	case Md5:
		hash = md5.New()
	case SHA1:
		hash = sha1.New()
	case SHA256:
		hash = sha256.New()
	case SHA512:
		hash = sha512.New()
	default:
		return "", errors.New(fmt.Sprintf("not support type: %v", hashType))
	}
	_, err := hash.Write([]byte(text))

	if err != nil {
		return "", err
	}
	if base64Encode {
		return base64.StdEncoding.EncodeToString(hash.Sum(nil)), nil
	} else {
		return hex.EncodeToString(hash.Sum(nil)), nil
	}
}

func HmacSign(hashType int, params, secret string, base64Encode bool) (string, error) {
	var mac hash.Hash
	switch hashType {
	case Md5:
		mac = hmac.New(md5.New, []byte(secret))
	case SHA1:
		mac = hmac.New(sha1.New, []byte(secret))
	case SHA256:
		mac = hmac.New(sha256.New, []byte(secret))
	case SHA512:
		mac = hmac.New(sha512.New, []byte(secret))
	default:
		return "", errors.New(fmt.Sprintf("not support type: %v", hashType))
	}

	_, err := mac.Write([]byte(params))
	if err != nil {
		return "", err
	}
	if base64Encode {
		return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
	} else {
		return hex.EncodeToString(mac.Sum(nil)), nil
	}
}

func AesEncrypt(encodeStr string, key, iv []byte) (string, error) {
	encodeBytes := []byte(encodeStr)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	blockSize := block.BlockSize()
	encodeBytes = PKCS7Padding(encodeBytes, blockSize)

	blockMode := cipher.NewCBCEncrypter(block, iv)
	crypted := make([]byte, len(encodeBytes))
	blockMode.CryptBlocks(crypted, encodeBytes)

	return hex.EncodeToString(crypted), nil
}
func AesDecrypt(decodeStr string, key, iv []byte) (string, error) {
	decodeBytes, err := hex.DecodeString(decodeStr)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	blockMode := cipher.NewCBCDecrypter(block, iv)
	origData := make([]byte, len(decodeBytes))
	blockMode.CryptBlocks(origData, []byte(decodeBytes))
	origData, err = PKCS7UnPadding(origData)
	if err != nil {
		return "", err
	}
	return string(origData), err
}

func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS7UnPadding(origData []byte) ([]byte, error) {
	length := len(origData)
	if length == 0 {
		return nil, errors.New("data can't be null！")
	} else {
		unpadding := int(origData[length-1])
		return origData[:(length - unpadding)], nil
	}
}
