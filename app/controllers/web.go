package controllers

import (
	"github.com/revel/revel"
	"strings"
	"io"
	"crypto/sha256"
	"crypto/aes"
	"crypto/rand"
	"crypto/cipher"
	"encoding/hex"
	"encoding/base64"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

const salt = "Yp2iD6PcTwB6upati0bPw314GrFWhUy90BIvbJTj5ETbbE8CoViDDGsJS6YHMOBq4VlwW3V00GWUMbbV"
const temp_transient_kek = "0ZugMhBCrbgdUcLeCTvN8QSjKnE8PHZsimDmXrkwpFRIDVrkGqJn061Bat8l34bcWROw0GEe3VtBifVy"

type Web struct {
	*revel.Controller
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

func (c Web) LoginForm() revel.Result {
	return c.Render()
}

func (c Web) LoginAction(username, password string) revel.Result {

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
		revel.TRACE.Println("Username and password not found")
	} else {
		
		// decrypt kek
		ps := []string{password, username, salt}
		key := []byte(string([]rune(strings.Join(ps, "-"))[0:32]))
		bdec, err := hex.DecodeString(result["encrypted_kek"])
		if err != nil {
			revel.TRACE.Println(err)
			return c.Redirect(App.Index)
		}
		kek := decrypt(key, bdec)
			
		revel.TRACE.Println("Login successful")
		revel.TRACE.Println(username)
		revel.TRACE.Println(kek)
						
		// redirect
		return c.Redirect(App.Index)
	}
						
	// redirect
	return c.Redirect(Web.LoginForm)
}

func (c Web) CreateUserForm() revel.Result {
	return c.Render()
}

func (c Web) CreateUserAction(username, password, confirm_password string) revel.Result {
	
	// hashed_password
	hashed_password := hashPassword(username, password)
	
	// encrypt kek
	ps := []string{password, username, salt}
	key := []byte(string([]rune(strings.Join(ps, "-"))[0:32]))
	pkek := []byte(temp_transient_kek)
	encrypted_kek := hex.EncodeToString(encrypt(key, pkek))
	
	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	
	// save user object
	dbu := session.DB("landline").C("Users")
	
	user_object_map := make(map[string]string)
	user_object_map["username"] = username
	user_object_map["hashed_password"] = hashed_password
	user_object_map["encrypted_kek"] = encrypted_kek
	
	err = dbu.Insert(user_object_map)
    if err != nil {
		panic(err)
    }

	// redirect
	return c.Redirect(Web.CreateUserForm)
}





// from: https://stackoverflow.com/questions/18817336/golang-encrypting-a-string-with-aes-and-base64

// See recommended IV creation from ciphertext below
//var iv = []byte{35, 46, 57, 24, 85, 35, 24, 74, 87, 35, 88, 98, 66, 32, 14, 05}

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

func decrypt(key, text []byte) string {
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
    return string(decodeBase64(string(text)))
}