package main

import (
	"encoding/json"
	"fmt"
	"os"
	"github.com/canopy-network/canopy/lib/crypto"
)

func main() {
	data, _ := os.ReadFile("/home/userland/.canopy/keystore.json")
	var ks struct {
		AddressMap map[string]*crypto.EncryptedPrivateKey
	}
	json.Unmarshal(data, &ks)
	addrs := []string{"205f68c279331cd17b9d41727f09eed7162b0389", "8f8b550064ec4ee4551d1666cb0ee5d35fc5154a"}
	for _, addr := range addrs {
		epk := ks.AddressMap[addr]
		if epk == nil { continue }
		pk, err := crypto.DecryptPrivateKey(epk, []byte("testpassword123"))
		if err != nil { fmt.Println(addr, "failed:", err); continue }
		fmt.Println("Address:", addr)
		fmt.Println("PublicKey:", epk.PublicKey)
		fmt.Println("PrivateKey:", pk.String())
	}
}
