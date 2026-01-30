package main

import "testing"

func TestFetchSymbolData(t *testing.T) {
	app := &application{}
	coinMarket, err := app.FetchSymbolData("bitcoin")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Fetched data: %+v", coinMarket)
}
