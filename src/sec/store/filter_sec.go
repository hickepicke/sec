package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Device struct {
	Addresses []string `json:"addresses"`
	Tags      []string `json:"tags"`
}

type Devices struct {
	Devices []Device `json:"devices"`
}

func main() {
	// Read the JSON file
	data, err := os.ReadFile("tailscale.json")
	if err != nil {
		panic(err)
	}

	var devices Devices
	if err := json.Unmarshal(data, &devices); err != nil {
		panic(err)
	}

	var result []string
	for _, d := range devices.Devices {
		for _, tag := range d.Tags {
			if tag == "tag:sec" && len(d.Addresses) > 0 {
				result = append(result, d.Addresses[0])
				break
			}
		}
	}

	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))
}
