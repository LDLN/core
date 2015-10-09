/*
 *  Copyright 2014-2015 LDLN
 *
 *  This file is part of LDLN Core.
 *
 *  LDLN Coret is free software: you can redistribute it and/or modify
 *  it under the terms of the GNU General Public License as published by
 *  the Free Software Foundation, either version 3 of the License, or
 *  any later version.
 *
 *  LDLN Core is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU General Public License for more details.
 *
 *  You should have received a copy of the GNU General Public License
 *  along with LDLN Core.  If not, see <http://www.gnu.org/licenses/>.
 */

package cryptoWrapper

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"fmt"
	"log"
	"github.com/RNCryptor/RNCryptor-go"
)

const salt = "Yp2iD6PcTwB6upati0bPw314GrFWhUy90BIvbJTj5ETbbE8CoViDDGsJS6YHMOBq4VlwW3V00GWUMbbV"

func HashPassword(username, password string) string {

	ps := []string{password, username, salt}

	// hashed_password
	hash := sha256.New()
	hash.Write([]byte(strings.Join(ps, "-")))
	md := hash.Sum(nil)
	hashed_password := hex.EncodeToString(md)

	return hashed_password
}

// from: https://stackoverflow.com/questions/18817336/golang-encrypting-a-string-with-aes-and-base64

// See recommended IV creation from ciphertext below
//var iv = []byte{35, 46, 57, 24, 85, 35, 24, 74, 87, 35, 88, 98, 66, 32, 14, 05}

func RandString(n int) string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}

func EncodeBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

func DecodeBase64(s string) []byte {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return data
}

//encrypt the text with the key provided
//returns a byte array
//For reference: http://crypto.stackexchange.com/questions/2476/cipher-feedback-mode
func Encrypt(key, text []byte) []byte {

  encrypted, err := rncryptor.Encrypt(string(key), text)
  if err != nil {
    log.Fatalln("error encrypting data: %v", err)
  } else {
    log.Println("encrypted: %v\n", string(encrypted))
  }
  
  return encrypted
}

//decrypt the text as a byte array with the key provided
//returns a byte array of the base64 decrypted text
func Decrypt(key, text []byte) []byte {
  fmt.Printf(string(key))
  decrypted, err := rncryptor.Decrypt(string(key), text)
  if err != nil {
		panic(err)
	} else {
    	log.Println("decrypted: %v\n", string(decrypted))
	} 

  return decrypted
}

