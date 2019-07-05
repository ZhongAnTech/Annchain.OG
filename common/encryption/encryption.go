// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
)

func createHash(key string) string {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

func encrypt(data []byte, passphrase string) (ciphertext []byte, err error) {
	block, _ := aes.NewCipher([]byte(createHash(passphrase)))
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return
	}
	ciphertext = gcm.Seal(nonce, nonce, data, nil)
	return
}

func decrypt(data []byte, passphrase string) (plaintext []byte, err error) {
	key := []byte(createHash(passphrase))
	block, err := aes.NewCipher(key)
	if err != nil {
		return
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err = gcm.Open(nil, nonce, ciphertext, nil)
	return
}

func EncryptFile(filename string, data []byte, passphrase string) (err error) {
	f, _ := os.Create(filename)
	defer f.Close()
	cipherText, err := encrypt(data, passphrase)
	if err != nil {
		return err
	}
	_, err = f.Write(cipherText)
	return
}

func DecryptFile(filename string, passphrase string) (data []byte, err error) {
	data, err = ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	return decrypt(data, passphrase)
}

func EncryptFileDummy(filename string, data []byte, passphrase string) (err error) {
	f, _ := os.Create(filename)
	defer f.Close()
	_, err = f.Write(data)
	return
}

func DecryptFileDummy(filename string, passphrase string) (data []byte, err error) {
	data, err = ioutil.ReadFile(filename)
	return
}
