package main

import (
	"fmt"
	"github.com/TeaOSLab/EdgeAPI/internal/dnsclients"
)

func main() {
	providerTypes := dnsclients.FindAllProviderTypes()
	for _, t := range providerTypes {
		fmt.Printf("Name: %s, Code: %s\n", t.GetString("name"), t.GetString("code"))
	}
}