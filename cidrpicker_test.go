package main

import (
	"fmt"
	"log"
	"net"
	"testing"
)

func TestFindBlock(t *testing.T) {
	subnets := make([]net.IPNet, 3)
	for i := 0; i < 3; i++ {
		_, sn, err := net.ParseCIDR(fmt.Sprintf("10.0.%v.0/24", i+1))
		if err != nil {
			log.Fatal(err)
		}
		subnets[i] = *sn
	}

	_, net, _ := net.ParseCIDR("10.0.0.0/10")

	block24 := FindBlock(*net, &subnets, 24)
	if block24.String() != "10.0.0.0/24" {
		t.Errorf("Found block %v, expected %v", block24.String(), "10.0.0.0/24")
	}

	block23 := FindBlock(*net, &subnets, 23)
	if block23.String() != "10.0.4.0/23" {
		t.Errorf("Found block %v, expected %v", block23.String(), "10.0.4.0/23")
	}

	block10 := FindBlock(*net, &subnets, 10)
	if block10.String() != "<nil>" {
		t.Errorf("Found block %v, expected %v", block10.String(), "<nil>")
	}
}
