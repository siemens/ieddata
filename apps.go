// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ieddata

import (
	"errors"
	"time"
)

// App describes an individual installed IE app, including version information.
// This is basically a “JOIN” of the “application” and “applicationversions”
// tables, using the appId column, of the platformbox.db.
//
// Field names almost always match their corresponding table column names, but
// with an initial uppercase letter necessary due to Go's implicit export rules.
// The few exceptions are URL and CompanyURL instead of Company/(w)ebAddress,
// Created/Modified instead of (c)reated/ModifiedDate, as well as avoiding
// stuttering as to not use “App” prefixes.
type App struct {
	Id                    string `db:"appId"`
	Version               string `db:"appVersion"`
	VersionId             string `db:"appVersionId"` // semantic version string
	VersionStatus         int
	ReleaseNotes          string
	OwnerId               string `db:"appOwnerId"`
	UserId                string
	ProjectId             string
	Title                 string
	RepositoryName        string
	Description           string
	URL                   string `db:"webAddress"`
	IconPath              string `db:"icon"`
	AppStatus             int
	CompanyName           string
	CompanyURL            string `db:"companyWebAddress"`
	IsDeveloperAppInstall int
	IsVisible             int
	SortWeight            int
	RunAsService          bool `db:"runasservice"`
	IsUpdatedOnPortal     int
	Created               time.Time `db:"createdDate"`
	Modified              time.Time `db:"modifiedDate"`
	ComposerFilepath      string    `db:"composerFilePath"`
	RedirectType          string
	RedirectUrl           string
	RESTRedirectUrl       string `db:"restRedirectUrl"`
	RedirectSection       string
	ToExecuteOrder        string
	Metadata              string
	ServiceLabels         string
	IsSecure              int
	IsSwarmModeEnable     int
	IsDebuggingEnabled    int `db:"isDebuggingEnabled"`
}

// Apps returns a slice of App elements with information about the currently
// installed apps and their versions. The information is read from the
// application and applicationversions tables in a “platformbox.db”, so make sure
// that the correct database has been Open'ed.
func (db *AppEngineDB) Apps() ([]App, error) {
	apps := make([]App, 0)
	// In order to not fail when there are new fields getting added, we need to
	// use the "unsafe" db handle here: for details, please see
	// https://jmoiron.github.io/sqlx/#safety.
	unsafedb := db.Unsafe()
	rows, err := unsafedb.Queryx("SELECT * FROM application INNER JOIN applicationversions USING(appId)")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var app App
		if err := rows.StructScan(&app); err != nil {
			return nil, err
		}
		if app.Id == "" {
			return nil, errors.New("empty IE App identifier: did you open the correct data base?")
		}
		apps = append(apps, app)
	}

	return apps, nil
}
