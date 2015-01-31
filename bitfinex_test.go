package bitfinex

import (
	"os"
	"strconv"
	"testing"
)

var APIKey = os.Getenv("BITFINEX_API_KEY")
var APISecret = os.Getenv("BITFINEX_API_SECRET")

var apiPublic = New("", "")
var apiPrivate = New(APIKey, APISecret)

func checkEnv(t *testing.T) {
	if APIKey == "" || APISecret == "" {
		t.Skip("Skipping test because because APIKey and/or APISecret env variables are not set")
	}
}

func TestTicker(t *testing.T) {
	// Test normal request
	ticker, err := apiPublic.Ticker("btcusd")
	if err != nil || ticker.LastPrice == 0 {
		t.Error("Failed: " + err.Error())
		return
	}

	// Test bad request,
	// which must return an error
	ticker, err = apiPublic.Ticker("random")
	if err == nil {
		t.Error("Failed")
		return
	}
}

func TestStats(t *testing.T) {
	// Test normal request
	stats, err := apiPublic.Stats("btcusd")
	if err != nil || len(stats) == 0 {
		t.Error("Failed: " + err.Error())
		return
	}

	// Test bad request,
	// which must return an error
	stats, err = apiPublic.Stats("random")
	if err == nil {
		t.Error("Failed")
		return
	}
}

func TestOrderbook(t *testing.T) {
	// Test normal request
	orderbook, err := apiPublic.Orderbook("btcusd", 2, 2, 1)
	if err != nil || len(orderbook.Asks) != 2 || len(orderbook.Bids) != 2 {
		t.Error("Failed: " + err.Error())
		return
	}
}

func TestLendbook(t *testing.T) {
	// Test normal request
	lendbook, err := apiPublic.Lendbook("btc", 2, 2)
	if err != nil || len(lendbook.Asks) != 2 || len(lendbook.Bids) != 2 {
		t.Error("Failed: " + err.Error())
		return
	}

	// Test bad request,
	// which must return an error
	lendbook, err = apiPublic.Lendbook("random", 2, 2)
	if err == nil {
		t.Error("Failed")
		return
	}
}

func TestWalletBalances(t *testing.T) {
	checkEnv(t)

	balances, err := apiPrivate.WalletBalances()
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}

	if len(balances) == 0 {
		t.Log("No wallet balances detected, please inspect")
		return
	}

	t.Log("Detected wallet balances, please inspect:")
	for k, v := range balances {
		t.Log("\t" + k.Type + ": " + strconv.FormatFloat(v.Amount, 'f', -1, 64) +
			" (available: " + strconv.FormatFloat(v.Available, 'f', -1, 64) + ") " + k.Currency)

	}

}

func TestNewOffer(t *testing.T) {
	checkEnv(t)

	offer, err := apiPrivate.NewOffer("BTC", 0.2, 365.0, 2, LEND)
	if err != nil || offer.ID == 0 {
		t.Error("Failed: " + err.Error())
		return
	}

	t.Log("Placed a new offer of 0.2BTC @ 1%/day for 2 days with ID: " + strconv.Itoa(offer.ID) + ", please inspect")
}

func TestActiveOffers(t *testing.T) {
	checkEnv(t)

	offers, err := apiPrivate.ActiveOffers()
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}

	if len(offers) == 0 {
		t.Log("No active offers detected, please inspect")
		return
	}

	t.Log("Detected active offers, please inspect:")
	for _, o := range offers {
		t.Log("\t" + strconv.Itoa(o.ID) + ": " + strconv.FormatFloat(o.OriginalAmount, 'f', -1, 64) +
			o.Currency + " @ " + strconv.FormatFloat(o.Rate/365., 'f', -1, 64) + "%/day for " + strconv.Itoa(o.Period) + " days")
	}

}

func TestActiveCredits(t *testing.T) {
	checkEnv(t)

	credits, err := apiPrivate.ActiveCredits()
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}

	if len(credits) == 0 {
		t.Log("No active credits found, please inspect")
		return
	}

	t.Log("Detected active credits, please inspect:")
	for _, c := range credits {
		t.Log("\t" + strconv.Itoa(c.ID) + ": " + strconv.FormatFloat(c.Amount, 'f', -1, 64) +
			c.Currency + " @ " + strconv.FormatFloat(c.Rate/365., 'f', -1, 64) + "%/day for " + strconv.Itoa(c.Period) + " days")
	}

}

func TestCancelOffer(t *testing.T) {
	checkEnv(t)

	// Assuming TestActiveOffers has PASSED
	offers, err := apiPrivate.ActiveOffers()
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}

	if len(offers) == 0 {
		t.Log("No active offers, nothing to cancel, please inspect")
		return
	}

	t.Log("Cancelling offer # " + strconv.Itoa(offers[0].ID))
	err = apiPrivate.CancelOffer(offers[0].ID)
	if err != nil {
		t.Error("Failed: " + err.Error())
		return
	}
}
