package main

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
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

func Encrypt(message string, e *big.Int, n *big.Int) string {
	//fmt.Println("The original message is: ", m, ", with bitwise length: ", m.BitLen())
	c := big.NewInt(0)
	m := stringToInt(message)
	c.Exp(m, e, n)
	//fmt.Println("The cipher is: ", c, ", with bitwise length: ", c.BitLen())
	return intToString(c)
}

func Decrypt(c *big.Int, d *big.Int, n *big.Int) *big.Int {
	m := big.NewInt(0)
	m.Exp(c, d, n)
	//fmt.Println("The decrypted message is: ", m, ", with bitwise length: ", m.BitLen())
	return m
}

func Hash(m *big.Int) *big.Int {
	s := sha256.Sum256(m.Bytes())
	r := big.NewInt(0)
	r.SetBytes(s[:])
	return r
}

func sign(message string, d *big.Int, n *big.Int) string {
	s := Hash(stringToInt(message))
	result := Encrypt(intToString(s), d, n)
	return result
}

func verify(m string, s string, e *big.Int, n *big.Int) bool {
	decryptedSign := Decrypt(stringToInt(s), e, n)
	hashedMessage := Hash(stringToInt(m))
	v := (decryptedSign.Cmp(hashedMessage) == 0)
	return v
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
