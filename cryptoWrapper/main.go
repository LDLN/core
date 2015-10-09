package cryptoWrapper

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"log"
	//	"crypto/rsa"
	"crypto/sha256"
	//	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"io"
	"strings"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
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
	if err != nil {
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

//encrypt the text with the key provided
//returns a byte array
//For reference: http://crypto.stackexchange.com/questions/2476/cipher-feedback-mode
func encrypt(key, text []byte) []byte {
	//Get a block using the AES and key to be run later
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	//Encode the text as base64
	b := encodeBase64(text)
	//Make an empty slice that is the length of the of the... aes BlockSize and the length of encoded b
	//Padding should be thought up here
	//(how the HELL do you figure the length of the encrypted data out on the other end?)
	//I guess you could take the aes block size (not sure which one it is) and subtract to get the rest?
	ciphertext := make([]byte, aes.BlockSize+len(b))
	//Get a reference (pointer) to the initiliazation vector as the first block of the empty ciphertext slice
	iv := ciphertext[:aes.BlockSize]
	//This creats a random number reader
	//Then it reads the random numbers into the iv block from above
	//The second parameter is a buffer to read into from the Reader, so it just fills the IV with random data
	//It returns the number of bytes read, then ignores that and checks if there's an error
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}

	//Do the actual encryption. Block is the AES block returned (opaque...)
	//IV is the random numbers from above
	//CFB is the resulting encrypted data Stream
	cfb := cipher.NewCFBEncrypter(block, iv)

	//So... for some reason they now XOR the encrypted data stream with the bytes of the base64 encoded plaintext
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(b))

	//According to https://play.golang.org/p/83dbz8bvYw
	//We should be running crypto/hmac on this before agreeing it's encrypted
	return ciphertext
}

func decrypt(key, text []byte) []byte {
	//Get a block using the AES and key to be run later
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	//Error out if the length of the byte array to be decrypted is less than a block size
	//This might have to be multiples of the block size if we do padding.
	if len(text) < aes.BlockSize {
		panic("ciphertext too short")
	}

	//Get the IV of the encryption from the first block size
	iv := text[:aes.BlockSize]
	//Get the encrypted text from the last block
	//If it's longer than a single block this would break in horrid ways
	text = text[aes.BlockSize:]

	//Decrypt the data into CFB using the AES block and IV we've set above
	cfb := cipher.NewCFBDecrypter(block, iv)

	//Again with the XOR.
	//This XOR's the decrypted data with the encrypted text
	//But in the encryption it was XOR'd with the original plaintext
	//HOW does this work?
	cfb.XORKeyStream(text, text)

	//return the base64 encoded string
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
