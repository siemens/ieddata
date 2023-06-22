// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ieddata_test

import (
	"fmt"

	"github.com/siemens/ieddata"
)

// Shows some of the IED key-values, such as the device name and owner name.
func Example_showDeviceInfo() {
	db, err := ieddata.Open("platformbox.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	di, err := db.DeviceInfo()
	if err != nil {
		panic(err)
	}

	fmt.Printf("device name: %s\nowner name: %s\n", di["deviceName"], di["ownerName"])
	// Output: device name: iedx12345
	// owner name: The Doctor
}
