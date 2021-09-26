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

var e *big.Int

func Keygen(k int) (*big.Int, *big.Int) {
	j := k / 2

	p := validPrime(j)
	q := validPrime(k - j)
	if p == nil || q == nil {
		return nil, nil
	}
	n := big.NewInt(0)
	n.Mul(p, q)
	g := big.NewInt(3)
	var z big.Int
	z.ModInverse(g, n)
	fmt.Println(z)
	return p, q
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

func Encrypt() {}
func Decrypt() {}

func main() {
	e = big.NewInt(3)
	p, q := Keygen(9)
	if p == nil || q == nil {
		return
	}
	fmt.Println(p)
	fmt.Println(q)
}
