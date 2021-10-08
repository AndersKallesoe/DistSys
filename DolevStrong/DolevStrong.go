package main

import "math/big"

type Party struct {
	id      int
	toypki  toyPKI
	relayed map[[2]string]bool //could potentially switch to bid, m struct
}

type toyPKI struct {
	pk  []*big.Int
	vks [][]*big.Int
}

func (t toyPKI) leak()      {}
func (t toyPKI) keygen(int) {}
func (p Party) Initialize() {}

func main() {}
