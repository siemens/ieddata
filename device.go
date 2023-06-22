// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ieddata

// DeviceInfo returns the key-value pairs describing an IED as per the device
// table in a platformbox.db.
func (db *AppEngineDB) DeviceInfo() (map[string]string, error) {
	rows, err := db.Query("SELECT deviceKey, deviceValue from device")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	devinfo := map[string]string{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		devinfo[key] = value
	}
	return devinfo, nil
}
