package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// k: n = pq
// e = 3 and d must satisfy that 3d mod (p−1)(q −1) = 1
// d = 3^{-1} mod(p-1)(q-1)
//gcd(3, p - 1) = gcd(3, q - 1) = 1

var e, d, n *big.Int

func Keygen(k int) {
	j := k / 2

	p := validPrime(j)
	q := validPrime(k - j)
	if p == nil || q == nil {
		return
	}
	fmt.Println("p = ", p, ", with bitwise length: ", p.BitLen())
	fmt.Println("q = ", q, ", with bitwise length: ", q.BitLen())
	n = big.NewInt(0)
	n.Mul(p, q)
	z := big.NewInt(0)
	d = big.NewInt(0)
	z.Mul(p.Sub(p, big.NewInt(1)), q.Sub(q, big.NewInt(1)))
	g := big.NewInt(3)
	d.ModInverse(g, z)
	fmt.Println("d = ", d, ", with bitwise length: ", d.BitLen())
	fmt.Println("n = ", n, ", with bitwise length: ", n.BitLen())

	return
}

/* finds a prime p of bitwise length k,
checks whether the GCD of (p-1) and e is 1 and if so returns p,
tries 100 times otherwise returns nil */
func validPrime(k int) *big.Int {
	var g, p *big.Int
	var err error
	for i := 0; i < 100; i++ {
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
	fmt.Println("could not find prime with length: ", k)
	return nil
}

func Encrypt(m *big.Int) *big.Int {
	fmt.Println("The original message is: ", m, ", with bitwise length: ", m.BitLen())
	c := big.NewInt(0)
	c.Exp(m, e, n)
	fmt.Println("The cipher is: ", c, ", with bitwise length: ", m.BitLen())
	return c
}
func Decrypt(c *big.Int) *big.Int {
	m := big.NewInt(0)
	m = m.Exp(c, d, n)
	fmt.Println("The decrypted message is: ", m, ", with bitwise length: ", m.BitLen())
	return m
}

func main() {
	e = big.NewInt(3)
	Keygen(13)
	Decrypt(Encrypt(big.NewInt(4253)))
}
