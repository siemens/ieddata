// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ieddata_test

import (
	"fmt"

	"golang.org/x/exp/slices"

	ieddata "github.com/siemens/ieddata"
)

// List the Industrial Edge Apps installed on this IED, with their titles and
// app IDs.
func Example_listInstalledApps() {
	db, err := ieddata.Open("platformbox.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	apps, err := db.Apps()
	if err != nil {
		panic(err)
	}

	// Sort the apps by their titles in order to get a stable output result that
	// can also be tested.
	slices.SortFunc(apps, func(a, b ieddata.App) bool { return a.Title < b.Title })
	for _, app := range apps {
		fmt.Printf("%q %s %s\n", app.Title, app.Version, app.Id)
	}
	// Output: "AppA" 1.9.18 195ff5e2e15a149ca5eb7c59d3857cc5
	// "AppB" 0.6.66666666666 7bd06d3bbf816d0658d5a871b0a498ff
	// "AppC" 1.1.0 1842f53281412f9c657c7765494ff80e
	// "AppD" 0.19.1 2a267358a0403fddb039924fbc4f3169
}
