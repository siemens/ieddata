// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ieddata

import (
	"database/sql"
	"errors"
	"reflect"
	"slices"
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
// stuttering as to not use “App” prefixes. Created and Modified are the dates
// of the application, not of its version.
type App struct {
	Id                    string `db:"appId"`
	Version               string `db:"appVersion"` // semantic version string
	VersionId             string `db:"appVersionId"`
	VersionStatus         int
	ReleaseNotes          string
	OwnerId               string `db:"appOwnerId"`
	UserId                string `db:"userId"`
	ProjectId             string `db:"projectId"`
	Title                 string `db:"title"`
	RepositoryName        string `db:"repositoryName"`
	Description           string `db:"description"`
	URL                   string `db:"webAddress"`
	IconPath              string `db:"icon"`
	AppStatus             int
	CompanyName           string `db:"companyName"`
	CompanyURL            string `db:"companyWebAddress"`
	IsDeveloperAppInstall int    `db:"isDeveloperAppInstall"`
	IsVisible             int    `db:"isVisible"`
	SortWeight            int    `db:"sortWeight"`
	RunAsService          bool   `db:"runasservice"`
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
	// Listing the application columns first lets them win over the
	// applicationversions columns of the same name, such as createdDate.
	rows, err := db.Query("SELECT application.*, applicationversions.* FROM application INNER JOIN applicationversions USING(appId)")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	// Map column indices to App field indices, with -1 for columns without a
	// field, so that columns added by newer IED versions don't break us.
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	appT := reflect.TypeFor[App]()
	columnFieldIndices := make([]int, len(cols))
	for columnIndex := range columnFieldIndices {
		columnFieldIndices[columnIndex] = -1 // no corresponding field
	}
	for fieldIdx := range appT.NumField() {
		columnName := appT.Field(fieldIdx).Tag.Get("db")
		if columnName == "" {
			columnName = FirstLower(appT.Field(fieldIdx).Name)
		}
		columnIdx := slices.Index(cols, columnName)
		if columnIdx < 0 {
			continue
		}
		columnFieldIndices[columnIdx] = fieldIdx
	}

	for rows.Next() {
		var app App
		appV := reflect.ValueOf(&app).Elem()
		values := make([]any, len(cols))
		for columnIdx := range values {
			fieldIndex := columnFieldIndices[columnIdx]
			if fieldIndex < 0 {
				values[columnIdx] = new(any)
				continue
			}
			values[columnIdx] = nullTolerant(appV.Field(fieldIndex).Addr().Interface())
		}
		if err := rows.Scan(values...); err != nil {
			return nil, err
		}

		if app.Id == "" {
			return nil, errors.New("empty IE App identifier: did you open the correct database?")
		}
		apps = append(apps, app)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return apps, nil
}

// nullable scans into dest, setting it to its zero value for NULL.
type nullable[T any] struct{ dest *T }

func (n nullable[T]) Scan(src any) error {
	var v sql.Null[T]
	if err := v.Scan(src); err != nil {
		return err
	}
	*n.dest = v.V
	return nil
}

// nullTolerant wraps the App field pointer dest so that scanning NULL into it
// doesn't fail.
func nullTolerant(dest any) any {
	switch d := dest.(type) {
	case *string:
		return nullable[string]{d}
	case *int:
		return nullable[int]{d}
	case *bool:
		return nullable[bool]{d}
	case *time.Time:
		return nullable[time.Time]{d}
	}
	return dest
}
