package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"regexp"
	"strings"
	"time"
)

/*
Methods from previous solution
*/

func Keygen(k int) (*big.Int, *big.Int, *big.Int) {
	var p, q *big.Int
	e := big.NewInt(3)
	for i := k / 2; i < k; i++ {
		p = validPrime(i, e)
		q = validPrime(k-i, e)
		if p != nil && q != nil && p.Cmp(q) != 0 {
			break
		}
	}
	if p == nil || q == nil {
		err := errors.New("Could not find required primes")
		panic(err)
	}

	n := big.NewInt(0)
	n.Mul(p, q)
	z := big.NewInt(0)
	d := big.NewInt(0)
	z.Mul(p.Sub(p, big.NewInt(1)), q.Sub(q, big.NewInt(1)))
	d.ModInverse(e, z)

	return d, e, n
}

func validPrime(k int, e *big.Int) *big.Int {
	var g, p *big.Int
	var err error
	for i := 0; i < 10; i++ {
		g = big.NewInt(3)
		p, err = rand.Prime(rand.Reader, k)
		if err != nil {
			fmt.Println("There was an error in creating a prime")
		}
		g.GCD(big.NewInt(1), big.NewInt(1), g.Sub(p, big.NewInt(1)), e)
		if g.Cmp(big.NewInt(1)) == 0 {
			return p
		}
	}

	return nil
}

func Encrypt(m *big.Int, e *big.Int, n *big.Int) *big.Int {
	c := big.NewInt(0)
	c.Exp(m, e, n)
	return c
}

func Decrypt(c *big.Int, d *big.Int, n *big.Int) *big.Int {
	m := big.NewInt(0)
	m.Exp(c, d, n)
	return m
}

func HashToInt(m *big.Int) *big.Int {
	s := sha256.Sum256(m.Bytes())
	r := big.NewInt(0)
	r.SetBytes(s[:])
	return r
}

func sign(m *big.Int, d *big.Int, n *big.Int) *big.Int {
	s := HashToInt(m)
	s = Encrypt(s, d, n)
	return s
}

func verify(m *big.Int, s *big.Int, e *big.Int, n *big.Int) bool {
	s = Decrypt(s, e, n)
	s2 := HashToInt(m)
	v := (s.Cmp(s2) == 0)
	return v
}

/*
new code
*/
func Generate(filename string, password string) string {
	if !SecurePassword(password) {
		return ""
	}
	d, e, n := Keygen(257)
	privatekey := KeyToString(d, n)
	EncryptKeyToFile(filename, password, privatekey)
	publickey := KeyToString(e, n)
	return publickey
}

func SecurePassword(password string) bool {
	if len(password) < 10 {
		fmt.Println("Error: password to short")
		return false
	}
	if match, _ := regexp.MatchString(`([a-z])`, password); !match {
		fmt.Println("Error: password must contain non capital letters")
		return false
	}
	if match, _ := regexp.MatchString(`([A-Z])`, password); !match {
		fmt.Println("Error: password must contain capital letters")
		return false
	}
	if match, _ := regexp.MatchString(`\d+`, password); !match {
		fmt.Println("Error: password must contain numbers")
		return false
	}
	return true
}

func Sign(filename string, password string, msg []byte) *big.Int {
	message := big.NewInt(0)
	if ValidateAndRemovePassword(filename, password) {
		s, m := DecryptKeyFromFile(filename, password)
		message.SetBytes(msg)
		fmt.Println("sign obtained")
		return sign(message, s, m)
	} else {
		fmt.Println("wrong password entered, timing out")
		time.Sleep(time.Second)
		fmt.Println("timeout over")
	}
	return message
}

//file should contain 2 lines. 1 with the secret key and one with the hash of the password.
func EncryptKeyToFile(filename string, password string, privatekey string) {
	hash := hashString(password)
	k := hash[:]
	hashedFileName := hashString(filename)
	iv := (hashedFileName[:])[0:16]
	message := []byte(privatekey)
	b, err := aes.NewCipher(k)

	if err != nil {
		panic("an error in creating block")
	}
	stream := cipher.NewCTR(b, iv[:])
	file, err := os.Create(filename)
	if err != nil {
		panic("an error in creating file")
	}
	cipherwriter := &cipher.StreamWriter{S: stream, W: file}
	cipherwriter.Write(message)
	ciphertext := make([]byte, aes.BlockSize+len(message))
	stream.XORKeyStream(ciphertext[aes.BlockSize:], message)
	cipherwriter.Close()
	WritePassword(filename, password)
}

func WritePassword(filename string, password string) {
	salt := hashString("salt")
	hash := hashString(password)
	saltedpw := hashString(string(salt[:]) + string(hash[:]))
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic("error in reopening file")
	}
	defer file.Close()
	if _, err := file.WriteString("\n" + string(saltedpw[:])); err != nil {
		panic("error in writing hash to file")
	}
}

func hashString(input string) [32]byte {
	cs256 := sha256.Sum256([]byte(input))
	return cs256
}

func DecryptKeyFromFile(filename string, password string) (*big.Int, *big.Int) {
	byteArray := make([]byte, 130)
	hash := hashString(password)
	k := hash[:]
	hashedFileName := hashString(filename)
	iv := (hashedFileName[:])[0:16]
	b, err := aes.NewCipher(k)
	if err != nil {
		panic("an error in creating block")
	}
	stream := cipher.NewCTR(b, iv)
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	reader := &cipher.StreamReader{S: stream, R: file}
	byteLen, err := reader.Read(byteArray)
	if err != nil {
		panic(err)
	}
	WritePassword(filename, password)
	return StringToKey(string(byteArray[:byteLen]))
}

func ValidateAndRemovePassword(filename string, password string) bool {
	salt := hashString("salt")
	hash := hashString(password)
	saltedpw := hashString(string(salt[:]) + string(hash[:]))
	hspw1 := saltedpw[:]

	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		panic("Error opening file!!!")
	}
	byteBuff := make([]byte, 128)
	totalLen, err := file.Read(byteBuff)
	if err != nil {
		fmt.Println(err)
	}
	c := byteBuff[:(totalLen - len(hspw1) - 1)]
	hspw2 := byteBuff[(totalLen - len(hspw1)):totalLen]
	if bytes.Equal(hspw1, hspw2) {
		os.Remove(filename)
		err := ioutil.WriteFile(filename, c, 0644)
		if err != nil {
			panic(err)
		}
		return true
	}
	return false

}

func stringToInt(s string) *big.Int {
	result := big.NewInt(0)
	result.SetString(s, 62)
	return result
}

func intToString(i *big.Int) string {
	result := i.Text(62)
	return result
}

func KeyToString(Key *big.Int, Modular *big.Int) string {
	return intToString(Key) + ":" + intToString(Modular)
}

func StringToKey(pk string) (*big.Int, *big.Int) {
	s := strings.Split(pk, ":")
	if len(s) != 2 {
		panic("Error: there was an error in splitting a public key")
	}
	return stringToInt(s[0]), stringToInt(s[1])
}

func test() {
	filename1 := "filename1"

	password1 := "password"
	password2 := "secretpassword"
	password3 := "SecretPassword"
	password4 := "14MaL363ND"
	password5 := "14MaL363NDs"

	message1 := []byte("this is a secret message")
	m1 := big.NewInt(0)
	m1.SetBytes(message1)

	fmt.Println("____________________________________________________________________")
	fmt.Println("Test of a password with which is to short: " + password1)
	Generate(filename1, password1)
	fmt.Println("____________________________________________________________________")
	fmt.Println("Test of a password with only non capital letters: " + password2)
	Generate(filename1, password2)
	fmt.Println("____________________________________________________________________")
	fmt.Println("Test of a password no numbers: " + password3)
	Generate(filename1, password3)
	fmt.Println("____________________________________________________________________")
	fmt.Println("Test that the generate method generates a public key:")
	p := Generate(filename1, password4)
	fmt.Print("public key: ")
	fmt.Println(p)
	fmt.Println("____________________________________________________________________")
	fmt.Println("Test that a signature cannot be obtained with the wrong password,")
	fmt.Println("and will cause a timeout: ")
	Sign(filename1, password5, message1)
	fmt.Println("____________________________________________________________________")
	fmt.Println("Test that a with the correct password and filename it is possible to")
	fmt.Println("obtain the sign for a message: ")
	signature := Sign(filename1, password4, message1)
	fmt.Println("____________________________________________________________________")
	fmt.Println("Test that the obtained signature can be verified:")
	v, m := StringToKey(p)
	fmt.Println(verify(m1, signature, v, m))
}

func main() {
	test()
}
