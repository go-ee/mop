// Copyright (c) 2013-2019 by Michael Dvorkin and contributors. All Rights Reserved.
// Use of this source code is governed by a MIT-style license that can
// be found in the LICENSE file.

package mop

import (
	"encoding/json"
	"io/ioutil"
	"sort"

	"github.com/Knetic/govaluate"
)

type Share struct {
	Trade float64
	Count int
}

// Profile manages Mop program settings as defined by user (ex. list of
// stock tickers). The settings are serialized using JSON and saved in
// the ~/.moprc file.
type Profile struct {
	Tickers       []string          // List of stock tickers to display.
	Shares        map[string]*Share // Ticker to share
	MarketRefresh int               // Time interval to refresh market data.
	QuotesRefresh int               // Time interval to refresh stock quotes.
	SortColumn    int               // Column number by which we sort stock quotes.
	Ascending     bool              // True when sort order is ascending.
	Grouped       bool              // True when stocks are grouped by advancing/declining.
	Filter        string            // Filter in human form
	ApiUrl        string            // API url of finance service
	ApiUrlParts   string            // API url parts for parameters

	tickersAll       []string                       //Tickers and Share Tickers
	filterExpression *govaluate.EvaluableExpression // The filter as a govaluate expression
	selectedColumn   int                            // Stores selected column number when the column editor is active.
	filename         string                         // Path to the file in which the configuration is stored
}

// Creates the profile and attempts to load the settings from ~/.moprc file.
// If the file is not there it gets created with default values.
func NewProfile(filename string, region string) *Profile {
	profile := &Profile{filename: filename}
	data, err := ioutil.ReadFile(filename)
	if err != nil { // Set default values:
		profile.MarketRefresh = 12 // Market data gets fetched every 12s (5 times per minute).
		profile.QuotesRefresh = 5  // Stock quotes get updated every 5s (12 times per minute).
		profile.Grouped = false    // Stock quotes are *not* grouped by advancing/declining.
		profile.Tickers = []string{`AMZ.F`, `GAZ.F`, `FB2A.F`, `ABEA.F`, `VOW3.F`, `AMD.F`, `LHA.F`, `TL0.F`, `AHLA.F`, `EBA.F`, `LUK.F`, `TUI1.F`, `AFR.F`, `MSF.F`, `INL.F`, `WDI.F`, `APC.F`, `5ZM.F`, `SCF.F`, `TU5A.F`, `BMW.F`, `IFX.F`, `PCE1.F`, `DAI.F`, `RY4D.F`, `EJT1.F`, `7HP.F`, `SAP.F`, `LHL.F`}
		profile.Shares = map[string]*Share{
			"AFR.F": &Share{
				Trade: 5.35,
				Count: 4000,
			},
			"LHL.F": &Share{
				Trade: 0.4863,
				Count: 30000,
			},
		}
		profile.SortColumn = 0   // Stock quotes are sorted by ticker name.
		profile.Ascending = true // A to Z.
		profile.Filter = ""
		profile.ApiUrl = `https://query1.finance.yahoo.com/v7/finance/quote?symbols=%s`
		if region == "de" {
			profile.ApiUrlParts = `&range=1d&interval=5m&indicators=close&includeTimestamps=false&includePrePost=false&region=DE&lang=de-DE&corsDomain=de.finance.yahoo.com&.tsrc=finance`
		} else {
			profile.ApiUrlParts = `&range=1d&interval=5m&indicators=close&includeTimestamps=false&includePrePost=false&corsDomain=finance.yahoo.com&.tsrc=finance`
		}
		profile.Save()
	} else {
		json.Unmarshal(data, profile)
		profile.SetFilter(profile.Filter)
	}
	profile.selectedColumn = -1
	profile.CalculateTickersAll()

	return profile
}

func (profile *Profile) CalculateTickersAll() {
	tickers := make(map[string]bool)
	for _, tracker := range profile.Tickers {
		tickers[tracker] = true
	}
	for ticker, _ := range profile.Shares {
		tickers[ticker] = true
	}
	profile.tickersAll = make([]string, 0)
	for tracker, _ := range tickers {
		profile.tickersAll = append(profile.tickersAll, tracker)
	}
}

// Save serializes settings using JSON and saves them in ~/.moprc file.
func (profile *Profile) Save() error {
	profile.CalculateTickersAll()

	data, err := json.Marshal(profile)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(profile.filename, data, 0644)
}

// AddTickers updates the list of existing tikers to add the new ones making
// sure there are no duplicates.
func (profile *Profile) AddTickers(tickers []string) (added int, err error) {
	added, err = 0, nil
	existing := make(map[string]bool)

	// Build a hash of existing tickers so we could look it up quickly.
	for _, ticker := range profile.Tickers {
		existing[ticker] = true
	}

	// Iterate over the list of new tickers excluding the ones that
	// already exist.
	for _, ticker := range tickers {
		if _, found := existing[ticker]; !found {
			profile.Tickers = append(profile.Tickers, ticker)
			added++
		}
	}

	if added > 0 {
		sort.Strings(profile.Tickers)
		err = profile.Save()
	}

	return
}

// RemoveTickers removes requested stock tickers from the list we track.
func (profile *Profile) RemoveTickers(tickers []string) (removed int, err error) {
	removed, err = 0, nil
	for _, ticker := range tickers {
		for i, existing := range profile.Tickers {
			if ticker == existing {
				// Requested ticker is there: remove i-th slice item.
				profile.Tickers = append(profile.Tickers[:i], profile.Tickers[i+1:]...)
				removed++
			}
		}
	}

	if removed > 0 {
		err = profile.Save()
	}

	return
}

// Reorder gets called by the column editor to either reverse sorting order
// for the current column, or to pick another sort column.
func (profile *Profile) Reorder() error {
	if profile.selectedColumn == profile.SortColumn {
		profile.Ascending = !profile.Ascending // Reverse sort order.
	} else {
		profile.SortColumn = profile.selectedColumn // Pick new sort column.
	}
	return profile.Save()
}

// Regroup flips the flag that controls whether the stock quotes are grouped
// by advancing/declining issues.
func (profile *Profile) Regroup() error {
	profile.Grouped = !profile.Grouped
	return profile.Save()
}

// SetFilter creates a govaluate.EvaluableExpression.
func (profile *Profile) SetFilter(filter string) {
	if len(filter) > 0 {
		var err error
		profile.filterExpression, err = govaluate.NewEvaluableExpression(filter)

		if err != nil {
			panic(err)
		}

	} else if len(filter) == 0 && profile.filterExpression != nil {
		profile.filterExpression = nil
	}

	profile.Filter = filter
	profile.Save()
}
