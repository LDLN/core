package cryptoWrapper

import (
	"log"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
//	"crypto/rsa"
	"crypto/sha256"
//	"crypto/x509"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"strings"
	"encoding/base64"
	"encoding/hex"
	"io"
)

const salt = "Yp2iD6PcTwB6upati0bPw314GrFWhUy90BIvbJTj5ETbbE8CoViDDGsJS6YHMOBq4VlwW3V00GWUMbbV"

func Encrypt(value, username, password string) string {
	
	// hashed_password
	hashed_password := hashPassword(username, password)

	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// find user object
	dbu := session.DB("landline").C("Users")
	var result map[string]string
	err = dbu.Find(bson.M{"username": username, "hashed_password": hashed_password}).One(&result)
	if(err != nil) {
		return ""
	}
	
	// decrypt kek
	ps := []string{password, username, salt}
	key := []byte(string([]rune(strings.Join(ps, "-"))[0:32]))
	bdec, err := hex.DecodeString(result["encrypted_kek"])
	if err != nil {
		log.Fatal(err)
	}
	dek := string(decrypt(key, bdec))

//	// decrypt rsa private
//	privenc, err := hex.DecodeString(result["encrypted_rsa_private"])
//	if err != nil {
//		log.Printf(err)
//	}
//	priva := decrypt(key, privenc)
//	priv, err := x509.ParsePKCS1PrivateKey(priva)
		
	return hex.EncodeToString(encrypt([]byte(dek), []byte(value)))
}

func hashPassword(username, password string) string {

	ps := []string{password, username, salt}

	// hashed_password
	hash := sha256.New()
	hash.Write([]byte(strings.Join(ps, "-")))
	md := hash.Sum(nil)
	hashed_password := hex.EncodeToString(md)

	return hashed_password
}

func encrypt(key, text []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	b := encodeBase64(text)
	ciphertext := make([]byte, aes.BlockSize+len(b))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(b))
	return ciphertext
}

func decrypt(key, text []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	if len(text) < aes.BlockSize {
		panic("ciphertext too short")
	}
	iv := text[:aes.BlockSize]
	text = text[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(text, text)
	return decodeBase64(string(text))
}

func encodeBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

func decodeBase64(s string) []byte {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return data
}