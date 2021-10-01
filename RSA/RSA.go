package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"os"
	"time"
)

// k: n = pq
// e = 3 and d must satisfy that 3d mod (p−1)(q −1) = 1
// d = 3^{-1} mod(p-1)(q-1)
//gcd(3, p - 1) = gcd(3, q - 1) = 1

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
	//fmt.Println("p = ", p, ", with bitwise length: ", p.BitLen())
	//fmt.Println("q = ", q, ", with bitwise length: ", q.BitLen())
	n := big.NewInt(0)
	n.Mul(p, q)
	z := big.NewInt(0)
	d := big.NewInt(0)
	z.Mul(p.Sub(p, big.NewInt(1)), q.Sub(q, big.NewInt(1)))
	d.ModInverse(e, z)
	//fmt.Println("d = ", d, ", with bitwise length: ", d.BitLen())
	//fmt.Println("n = ", n, ", with bitwise length: ", n.BitLen())

	return d, e, n
}

/* finds a prime p of bitwise length k,
checks whether the GCD of (p-1) and e is 1 and if so returns p,
tries 100 times otherwise returns nil */
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
	//fmt.Println("could not find valid prime with length: ", k)
	return nil
}

func Encrypt(m *big.Int, e *big.Int, n *big.Int) *big.Int {
	//fmt.Println("The original message is: ", m, ", with bitwise length: ", m.BitLen())
	c := big.NewInt(0)
	c.Exp(m, e, n)
	//fmt.Println("The cipher is: ", c, ", with bitwise length: ", c.BitLen())
	return c
}

func Decrypt(c *big.Int, d *big.Int, n *big.Int) *big.Int {
	m := big.NewInt(0)
	m.Exp(c, d, n)
	//fmt.Println("The decrypted message is: ", m, ", with bitwise length: ", m.BitLen())
	return m
}

// k must be 16, 24 or 32 bytes
func EncryptToFile(k []byte, m *big.Int, fileName string) []byte {
	message := m.FillBytes(make([]byte, 8))
	b, err := aes.NewCipher(k)
	if err != nil {
		fmt.Println("an error in creating block")
	}
	iv := make([]byte, b.BlockSize())
	rand.Read(iv)
	stream := cipher.NewCTR(b, iv)
	file, err := os.Create(fileName)
	writer := &cipher.StreamWriter{S: stream, W: file}
	writer.Write(message)
	ciphertext := make([]byte, aes.BlockSize+len(message))
	stream.XORKeyStream(ciphertext[aes.BlockSize:], message)
	writer.Close()
	return iv
}

func DecryptFromFile(k []byte, iv []byte, filename string) uint64 {
	byteArray := make([]byte, 8)
	b, err := aes.NewCipher(k)
	if err != nil {
		panic(err)
	}
	stream := cipher.NewCTR(b, iv)

	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	reader := &cipher.StreamReader{S: stream, R: file}
	_, err = reader.Read(byteArray)
	if err != nil {
		panic(err)
	}
	result := binary.BigEndian.Uint64(byteArray[:])
	//fmt.Println(result)
	return result
}

func testKeygen() {
	for i := 0; i < 1000; i++ {
		testnr, err := rand.Int(rand.Reader, big.NewInt(1000000000000000000))
		if err != nil {
			panic(err)
		}
		d, e, n := Keygen(testnr.BitLen() + 1)
		result := Decrypt(Encrypt(testnr, e, n), d, n)
		if testnr.Cmp(result) != 0 {
			fmt.Println("failed for testnr=", testnr, "result=", result)
		}
		if n.Cmp(big.NewInt(int64(testnr.BitLen()))) < 0 {
			fmt.Println("there was a problem´with the modulus")
		}
	}
	fmt.Println("if any random number between 0 and 10^18 failed it will be printed above")
}

func testEncryptToFile() {
	for i := 0; i < 1000; i++ {
		key := make([]byte, 32)
		rand.Read(key)
		testnr, err := rand.Int(rand.Reader, big.NewInt(1000000000000000000))
		if err != nil {
			panic(err)
		}
		d, e, n := Keygen(testnr.BitLen() + 1)
		iv := EncryptToFile(key, d, "test.txt")
		d = big.NewInt(int64(DecryptFromFile(key, iv, "test.txt")))
		result := Decrypt(Encrypt(testnr, e, n), d, n)
		if testnr.Cmp(result) != 0 {
			fmt.Println("failed for testnr=", testnr, "result=", result)
		}
	}
	fmt.Println("if any random number between 0 and 10^18 failed it will be printed above")
}

func Hash(m *big.Int) *big.Int {
	s := sha256.Sum256(m.Bytes())
	r := big.NewInt(0)
	r.SetBytes(s[:])
	return r
}

func sign(m *big.Int, d *big.Int, n *big.Int) *big.Int {
	s := Hash(m)
	s = Encrypt(s, d, n)
	return s
}

func verify(m *big.Int, s *big.Int, e *big.Int, n *big.Int) bool {
	s = Decrypt(s, e, n)
	s2 := Hash(m)
	v := (s.Cmp(s2) == 0)
	return v
}

func TestSignAndVerify() {

	m := big.NewInt(42)
	fmt.Println("m:= 42")
	d, e, n := Keygen(256)
	fmt.Println("RSA public and private key and modular was generated: ")
	fmt.Println("d,e,n=", d, e, n)
	s := sign(m, d, n)
	fmt.Println("the sign of m was: ", s)
	v := verify(m, s, e, n)
	fmt.Println("the sign was decrypted, verified and found to be:", v, "for m = ", m)
	w := big.NewInt(24)
	fmt.Println("w:= 24")
	v = verify(w, s, e, n)
	fmt.Println("the sign was decrypted, verified and found to be:", v, "for w = ", w)
}

func TestHashSpeed() {
	//generate 10000 random bytes
	m := make([]byte, 100000)
	_, err := rand.Read(m)
	if err != nil {
		panic(err)
	}
	// turn the bytearray into a number
	n := big.NewInt(0)
	n.SetBytes(m)
	//record starttime
	start := time.Now()
	fmt.Println("beginning hashing")
	//hash the number
	h := Hash(n)
	//record finish time
	t := time.Now()
	elapsed := t.Sub(start)
	fmt.Println("finished hash", h)
	fmt.Println("100 kB hash in :", elapsed.Microseconds(), "microSeconds")
	s := (100000 * 1000000) / elapsed.Microseconds()
	fmt.Println("this means that the speed of the hash was: ", s, "in bit/s")
}

func TestRSASpeed() {
	d, _, n := Keygen(2001)
	m := big.NewInt(42)
	m = Hash(m)
	start := time.Now()
	fmt.Println("beginning encryption")
	s := Encrypt(m, d, n)
	//record finish time
	t := time.Now()
	elapsed := t.Sub(start)
	fmt.Println("finished encryption", s)
	fmt.Println("encypted sha-256 hash in :", elapsed.Microseconds(), "microSeconds")
}

func main() {
	//TestSignAndVerify()
	//TestHashSpeed()
	TestRSASpeed()
}
