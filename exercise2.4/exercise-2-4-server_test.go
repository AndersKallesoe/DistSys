package main

import (
	"fmt"
	"testing"
)

func TestGetIPandPort(t *testing.T) {
	fmt.Println("Here")
	n := mkNode()
	s := n.GetIPandPort()
	fmt.Println(s)
}

func TestSend(t *testing.T) {

}

func TestListen(t *testing.T) {

}

func TestHandleConnection(t *testing.T) {

}

func TestMakeConns(t *testing.T) {

}

func TestPrintHostNames(t *testing.T) {

}

func TestPrintMap(t *testing.T) {

}
