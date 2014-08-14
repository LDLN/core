package controllers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"github.com/revel/revel"
	"io"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"strings"
	"github.com/nu7hatch/gouuid"
)

const salt = "Yp2iD6PcTwB6upati0bPw314GrFWhUy90BIvbJTj5ETbbE8CoViDDGsJS6YHMOBq4VlwW3V00GWUMbbV"

type Web struct {
	*revel.Controller
}

func checkIfSetupIsEligible() bool {

	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// find any deployments
	dbd := session.DB("landline").C("Deployments")
	var resultd map[string]string
	err = dbd.Find(bson.M{}).One(&resultd)

	// find any users
	dbu := session.DB("landline").C("Users")
	var resultu map[string]string
	err = dbu.Find(bson.M{}).One(&resultu)
	
	// cannot do setup if users exist
	if err != nil {
		return true
	} else {
		return false
	}
}

func (c Web) FirstTimeSetupForm() revel.Result {

	if(!checkIfSetupIsEligible()) {
		c.Flash.Error("Basestation is already setup")
		return c.Redirect(Web.LoginForm)
	}
	
	return c.Render()
}

func (c Web) FirstTimeSetupAction(org_title, org_subtitle, org_mbtiles_file, org_map_center_lat, org_map_center_lon, org_map_zoom_min, org_map_zoom_max, username, password, confirm_password string) revel.Result {

	if(!checkIfSetupIsEligible()) {
		c.Flash.Error("Basestation is already setup")
		return c.Redirect(Web.LoginForm)
	}
	
	// create deployment
	if(createDeployment(org_title, org_subtitle, org_mbtiles_file, org_map_center_lat, org_map_center_lon, org_map_zoom_min, org_map_zoom_max)) {
		
		// create new key for organization
		skek := randString(32)
		
		// create first user account
		if(createUser(username, password, skek)) {
			c.Flash.Success("Organization and user created")
		} else {
			c.Flash.Error("Error generating user")
		}
		
	} else {
		c.Flash.Error("Error creating organization")
	}
	
	return c.Redirect(Web.LoginForm)
}

func createDeployment(org_title, org_subtitle, org_mbtiles_file, org_map_center_lat, org_map_center_lon, org_map_zoom_min, org_map_zoom_max string) bool {
	
	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// save user object
	dbu := session.DB("landline").C("Deployments")

	// create object
	deployment_object_map := make(map[string]string)
	uuid, err := uuid.NewV4()
	deployment_object_map["uuid"] = uuid.String()
	deployment_object_map["name"] = org_title
	deployment_object_map["unit"] = org_subtitle
	deployment_object_map["map_center_lat"] = org_map_center_lat
	deployment_object_map["map_center_lon"] = org_map_center_lon
	deployment_object_map["map_zoom_min"] = org_map_zoom_min
	deployment_object_map["map_zoom_max"] = org_map_zoom_max
	deployment_object_map["map_mbtiles"] = org_mbtiles_file

	err = dbu.Insert(deployment_object_map)
	if err != nil {
		panic(err)
	}
	
	return true;
}

func (c Web) requireAuth() bool {
	if(c.Session["username"] == "" || c.Session["kek"] == "") {
		revel.TRACE.Println("User not authd")
		return false
	}
	revel.TRACE.Println("User authd")
	return true
}

func (c Web) WebSocketTest() revel.Result {
	return c.Render()
}

func (c Web) Logout() revel.Result {
	c.Session["username"] = ""
	c.Session["kek"] = ""
	c.Flash.Success("You have logged out successfully")
	return c.Redirect(Web.LoginForm)
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
	
	if(checkIfSetupIsEligible()) {
		return c.Redirect(Web.FirstTimeSetupForm)
	}
	
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
			return c.Redirect(Web.LoginForm)
		}
		kek := string(decrypt(key, bdec))

		// decrypt rsa private
		privenc, err := hex.DecodeString(result["encrypted_rsa_private"])
		if err != nil {
			revel.TRACE.Println(err)
			return c.Redirect(Web.LoginForm)
		}
		priva := decrypt(key, privenc)
		priv, err := x509.ParsePKCS1PrivateKey(priva)

		revel.TRACE.Println("Login successful")
		revel.TRACE.Println(username)
		revel.TRACE.Println(kek)
		revel.TRACE.Println(priv)
		
		// get deployment
		dbd := session.DB("landline").C("Deployments")
		var resultd map[string]string
		err = dbd.Find(bson.M{}).One(&resultd)
		
		// save to session
		c.Session["kek"] = kek;
		c.Session["username"] = username;
		c.Session["deployment_name"] = resultd["name"];
		c.Session["deployment_unit"] = resultd["unit"];

		// redirect
		return c.Redirect(Web.WebSocketTest)
	}

	// redirect
	c.Flash.Error("Username and password not found")
	return c.Redirect(Web.LoginForm)
}

func (c Web) CreateUserForm() revel.Result {
	if(!c.requireAuth()) {
		return c.Redirect(Web.LoginForm)
	}
	return c.Render()
}

func (c Web) CreateUserAction(username, password, confirm_password string) revel.Result {
	
	if(!c.requireAuth()) {
		return c.Redirect(Web.LoginForm)
	}

	// get kek
	var skek string
	if(c.Session["kek"] == "") {
		c.Flash.Error("Error generating user")
		return c.Redirect(Web.CreateUserForm)
	} else {
		skek = c.Session["kek"];
	}
	
	// create user
	if(createUser(username, password, skek)) {
		c.Flash.Success("User created")
	} else {
		c.Flash.Error("Error generating user")
	}

	// redirect
	return c.Redirect(Web.CreateUserForm)
}

func createUser(username, password, skek string) bool {
	
	// hashed_password
	hashed_password := hashPassword(username, password)

	// encrypt kek
	ps := []string{password, username, salt}
	key := []byte(string([]rune(strings.Join(ps, "-"))[0:32]))
	pkek := []byte(skek)
	encrypted_kek := hex.EncodeToString(encrypt(key, pkek))

	// generate rsa keypair for user
	size := 1024
	priv, err := rsa.GenerateKey(rand.Reader, size)
	if err != nil {
		revel.TRACE.Println("failed to generate key")
	}
	if bits := priv.N.BitLen(); bits != size {
		revel.TRACE.Println("key too short (%d vs %d)", bits, size)
	}
	pub, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	rsa_public_string := hex.EncodeToString(pub)

	revel.TRACE.Println(priv)

	// encrypt rsa private keypair
	encrypted_rsa_private := hex.EncodeToString(encrypt(key, x509.MarshalPKCS1PrivateKey(priv)))

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
	user_object_map["encrypted_rsa_private"] = encrypted_rsa_private
	user_object_map["rsa_public"] = rsa_public_string

	err = dbu.Insert(user_object_map)
	if err != nil {
		panic(err)
	}
	
	return true;
}

// from: https://stackoverflow.com/questions/18817336/golang-encrypting-a-string-with-aes-and-base64

// See recommended IV creation from ciphertext below
//var iv = []byte{35, 46, 57, 24, 85, 35, 24, 74, 87, 35, 88, 98, 66, 32, 14, 05}

func randString(n int) string {
    const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
    var bytes = make([]byte, n)
    rand.Read(bytes)
    for i, b := range bytes {
        bytes[i] = alphanum[b % byte(len(alphanum))]
    }
    return string(bytes)
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
