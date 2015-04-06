// TODO: Public: Orderbook, Trades, Lends, Symbols, Symbols Details
// TODO: Authenticated: New deposit, New order, Multiple new orders, Cancel order, Cancel multiple orders, Cancel all active orders, Replace order, Order status, Active Orders, Active Positions, Claim position, Past trades, Offer status, Active Swaps used in a margin position, Balance history, Close swap, Account informations, Margin informations

package bitfinex

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// APIURL points to Bitfinex API URL, found at https://www.bitfinex.com/pages/API
	APIURL = "https://api.bitfinex.com"
	// LEND ...
	LEND = "lend"
	// BORROW ...
	BORROW = "borrow"
)

// API structure stores Bitfinex API credentials
type API struct {
	APIKey    string
	APISecret string
}

// ErrorMessage ...
type ErrorMessage struct {
	Message string `json:"message"` // Returned only on error
}

// Ticker ...
type Ticker struct {
	Mid       float64 `json:"mid,string"`        // mid (price): (bid + ask) / 2
	Bid       float64 `json:"bid,string"`        // bid (price): Innermost bid.
	Ask       float64 `json:"ask,string"`        // ask (price): Innermost ask.
	LastPrice float64 `json:"last_price,string"` // last_price (price) The price at which the last order executed.
	Low       float64 `json:"low,string"`        // low (price): Lowest trade price of the last 24 hours
	High      float64 `json:"high,string"`       // high (price): Highest trade price of the last 24 hours
	Volume    float64 `json:"volume,string"`     // volume (price): Trading volume of the last 24 hours
	Timestamp float64 `json:"timestamp,string"`  // timestamp (time) The timestamp at which this information was valid.
}

// Stats ...
type Stats []Stat

// Stat ...
type Stat struct {
	Period int     `json:"period"`        // period (integer), period covered in days
	Volume float64 `json:"volume,string"` // volume (price)
}

// Lendbook ...
type Lendbook struct {
	Bids []LendbookOffer // bids (array of loan demands)
	Asks []LendbookOffer // asks (array of loan offers)
}

// Orderbook ... Public (NEW)
type Orderbook struct {
	Bids []OrderbookOffer // bids (array of bid offers)
	Asks []OrderbookOffer // asks (array of ask offers)
}

// OrderbookOffer ... (NEW)
type OrderbookOffer struct {
	Price     float64 `json:"price,string"`     // price
	Amount    float64 `json:"amount,string"`    // amount (decimal)
	Timestamp float64 `json:"timestamp,string"` // time
}

// LendbookOffer ...
type LendbookOffer struct {
	Rate      float64 `json:"rate,string"`      // rate (rate in % per 365 days)
	Amount    float64 `json:"amount,string"`    // amount (decimal)
	Period    int     `json:"period"`           // period (days): minimum period for the loan
	Timestamp float64 `json:"timestamp,string"` // timestamp (time)
	FRRString string  `json:"frr"`              // frr (yes/no): "Yes" if the offer is at Flash Return Rate, "No" if the offer is at fixed rate
	FRR       bool
}

// WalletBalance ...
type WalletBalance struct {
	Type      string  `json:"type"`             // "trading", "deposit" or "exchange".
	Currency  string  `json:"currency"`         // Currency
	Amount    float64 `json:"amount,string"`    // How much balance of this currency in this wallet
	Available float64 `json:"available,string"` // How much X there is in this wallet that is available to trade.
}

// WalletKey ...
type WalletKey struct {
	Type, Currency string
}

// WalletBalances ...
type WalletBalances map[WalletKey]WalletBalance

// MyTrades ... (NEW)
type MyTrades []MyTrade

// MyTrade ... (NEW)
type MyTrade struct {
	Price       float64 `json:"price,string"`      // price
	Amount      float64 `json:"amount,string"`     // amount (decimal)
	Timestamp   float64 `json:"timestamp,string"`  // time
	Until       float64 `json:"until,string"`      // until (time): return only trades before or a the time specified here
	Exchange    string  `json:"exchange"`          // exchange
	Type        string  `json:"type"`              // type - "Sell" or "Buy"
	FeeCurrency string  `json:"fee_currency"`      // fee_currency (string) Currency you paid this trade's fee in
	FeeAmount   float64 `json:"fee_amount,string"` // fee_amount (decimal) Amount of fees you paid for this trade
	TID         int     `json:"tid"`               // tid (integer): unique identification number of the trade
	OrderId     int     `json:"order_id"`          // order_id (integer) unique identification number of the parent order of the trade
}

// Offer ...
type Offer struct {
	ID              int     `json:"id"`
	Currency        string  `json:"currency"`                // The currency name of the offer.
	Rate            float64 `json:"rate,string"`             // The rate the offer was issued at (in % per 365 days).
	Period          int     `json:"period"`                  // The number of days of the offer.
	Direction       string  `json:"direction"`               // Either "lend" or "loan".Either "lend" or "loan".
	Type            string  `json:"type"`                    // Either "market" / "limit" / "stop" / "trailing-stop".
	Timestamp       float64 `json:"timestamp,string"`        // The timestamp the offer was submitted.
	Live            bool    `json:"is_live,bool"`            // Could the offer still be filled?
	Cacelled        bool    `json:"is_cancelled,bool"`       // Has the offer been cancelled?
	ExecutedAmount  float64 `json:"executed_amount,string"`  // How much of the offer has been executed so far in its history?
	RemainingAmount float64 `json:"remaining_amount,string"` // How much is still remaining to be submitted?
	OriginalAmount  float64 `json:"original_amount,string"`  // What was the offer originally submitted for?
}

// Offers ...
type Offers []Offer

// Credit ...
type Credit struct {
	ID        int     `json:"id"`
	Currency  string  `json:"currency"`         // The currency name of the offer.
	Rate      float64 `json:"rate,string"`      // The rate the offer was issued at (in % per 365 days).
	Period    int     `json:"period"`           // The number of days of the offer.
	Amount    float64 `json:"amount,string"`    // How much is the credit for
	Status    string  `json:"status"`           // "Active"
	Timestamp float64 `json:"timestamp,string"` // The timestamp the offer was submitted.

}

// Credits ...
type Credits []Credit

// New returns a new Bitfinex API instance
func New(key, secret string) (api *API) {
	api = &API{
		APIKey:    key,
		APISecret: secret,
	}
	return api
}

///////////////////////////////////////
// Main API methods
///////////////////////////////////////

// Ticker returns innermost bid and asks and information on the most recent trade,
//	as well as high, low and volume of the last 24 hours.
func (api *API) Ticker(symbol string) (ticker Ticker, err error) {
	symbol = strings.ToLower(symbol)

	body, err := api.get("/v1/ticker/" + symbol)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &ticker)
	if err != nil || ticker.LastPrice == 0 { // Failed to unmarshal expected message
		// Attempt to unmarshal the error message
		errorMessage := ErrorMessage{}
		err = json.Unmarshal(body, &errorMessage)
		if err != nil { // Not expected message and not expected error, bailing...
			return
		}

		return ticker, errors.New("API: " + errorMessage.Message)
	}

	return
}

// Stats return various statistics about the requested pairs.
func (api *API) Stats(symbol string) (stats Stats, err error) {
	symbol = strings.ToLower(symbol)

	body, err := api.get("/v1/stats/" + symbol)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &stats)
	if err != nil || len(stats) == 0 { // Failed to unmarshal expected message
		// Attempt to unmarshal the error message
		errorMessage := ErrorMessage{}
		err = json.Unmarshal(body, &errorMessage)
		if err != nil { // Not expected message and not expected error, bailing...
			return
		}

		return stats, errors.New("API: " + errorMessage.Message)
	}

	return
}

// Orderbook returns the full order book.
func (api *API) Orderbook(symbol string, limitBids, limitAsks, group int) (orderbook Orderbook, err error) {
	symbol = strings.ToLower(symbol)

	body, err := api.get("/v1/book/" + symbol + "?limit_bids=" + strconv.Itoa(limitBids) + "&limit_asks=" + strconv.Itoa(limitAsks) + "&group=" + strconv.Itoa(group))
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &orderbook)
	if err != nil {
		return
	}

	return
}

// Lendbook returns the full lend book.
func (api *API) Lendbook(currency string, limitBids, limitAsks int) (lendbook Lendbook, err error) {
	currency = strings.ToLower(currency)

	body, err := api.get("/v1/lendbook/" + currency + "?limit_bids=" + strconv.Itoa(limitBids) + "&limit_asks=" + strconv.Itoa(limitAsks))
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &lendbook)
	if err != nil {
		return
	}

	if (limitAsks != 0 && len(lendbook.Asks) == 0) || (limitBids != 0 && len(lendbook.Bids) == 0) {
		return lendbook, errors.New("API: Lendbook empty, likely bad currency specified")
	}

	// Convert FRR strings to boolean values
	for _, p := range [](*[]LendbookOffer){&lendbook.Asks, &lendbook.Bids} {
		for i, e := range *p {
			if strings.ToLower(e.FRRString) == "yes" {
				e.FRR = true
				(*p)[i] = e
			}
		}
	}

	return
}

// WalletBalances return your balances.
func (api *API) WalletBalances() (wallet WalletBalances, err error) {
	request := struct {
		URL   string `json:"request"`
		Nonce string `json:"nonce"`
	}{
		"/v1/balances",
		strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	body, err := api.post(request.URL, request)
	if err != nil {
		return
	}

	tmpBalances := []WalletBalance{}
	err = json.Unmarshal(body, &tmpBalances)
	if err != nil { // Failed to unmarshal expected message
		// Attempt to unmarshal the error message
		errorMessage := ErrorMessage{}
		err = json.Unmarshal(body, &errorMessage)
		if err != nil { // Not expected message and not expected error, bailing...
			return
		}

		return nil, errors.New("API: " + errorMessage.Message)
	}

	wallet = make(WalletBalances)
	for _, w := range tmpBalances {
		wallet[WalletKey{w.Type, w.Currency}] = w
	}

	return
}

// MyTrades returns an array of your past trades for the given symbol.
func (api *API) MyTrades(symbol string, timestamp string, limitTrades int) (mytrades MyTrades, err error) {
	symbol = strings.ToLower(symbol)

	request := struct {
		URL         string `json:"request"`
		Nonce       string `json:"nonce"`
		Symbol      string `json:"symbol"`
		Timestamp   string `json:"timestamp"`
		LimitTrades int    `json:"limit_trades"`
	}{
		URL:         "/v1/mytrades",
		Nonce:       strconv.FormatInt(time.Now().UnixNano(), 10),
		Symbol:      symbol,
		Timestamp:   timestamp,
		LimitTrades: limitTrades,
	}

	body, err := api.post(request.URL, request)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &mytrades)
	if err != nil { // Failed to unmarshal expected message
		// Attempt to unmarshal the error message
		errorMessage := ErrorMessage{}
		err = json.Unmarshal(body, &errorMessage)
		if err != nil { // Not expected message and not expected error, bailing...
			return
		}

		return nil, errors.New("API: " + errorMessage.Message)
	}
	return
}

// CancelOffer cancel an offer give its id.
func (api *API) CancelOffer(id int) (err error) {
	request := struct {
		URL     string `json:"request"`
		Nonce   string `json:"nonce"`
		OfferID int    `json:"offer_id"`
	}{
		"/v1/offer/cancel",
		strconv.FormatInt(time.Now().UnixNano(), 10),
		id,
	}

	body, err := api.post(request.URL, request)
	if err != nil {
		return
	}

	tmpOffer := struct {
		ID        int  `json:"id"`
		Cancelled bool `json:"is_cancelled,bool"`
	}{}

	err = json.Unmarshal(body, &tmpOffer)
	if err != nil || tmpOffer.ID != id { // Failed to unmarshal expected message
		// Attempt to unmarshal the error message
		errorMessage := ErrorMessage{}
		err = json.Unmarshal(body, &errorMessage)
		if err != nil { // Not expected message and not expected error, bailing...
			return
		}

		return errors.New("API: " + errorMessage.Message)
	}

	if tmpOffer.Cancelled == true {
		return errors.New("API: Offer already cancelled")
	}

	return
}

// ActiveCredits return a list of currently lent funds (active credits).
func (api *API) ActiveCredits() (credits Credits, err error) {
	request := struct {
		URL   string `json:"request"`
		Nonce string `json:"nonce"`
	}{
		"/v1/credits",
		strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	body, err := api.post(request.URL, request)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &credits)
	if err != nil { // Failed to unmarshal expected message
		// Attempt to unmarshal the error message
		errorMessage := ErrorMessage{}
		err = json.Unmarshal(body, &errorMessage)
		if err != nil { // Not expected message and not expected error, bailing...
			return
		}

		return credits, errors.New("API: " + errorMessage.Message)
	}

	return
}

// ActiveOffers return an array of all your live offers (lending or borrowing).
func (api *API) ActiveOffers() (offers Offers, err error) {
	request := struct {
		URL   string `json:"request"`
		Nonce string `json:"nonce"`
	}{
		"/v1/offers",
		strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	body, err := api.post(request.URL, request)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &offers)
	if err != nil { // Failed to unmarshal expected message
		// Attempt to unmarshal the error message
		errorMessage := ErrorMessage{}
		err = json.Unmarshal(body, &errorMessage)
		if err != nil { // Not expected message and not expected error, bailing...
			return
		}

		return offers, errors.New("API: " + errorMessage.Message)
	}

	return
}

// NewOffer submits a new offer.
// currency (string): The name of the currency.
// amount (decimal): Offer size: how much to lend or borrow.
// rate (decimal): Rate to lend or borrow at. In percentage per 365 days.
// period (integer): Number of days of the loan (in days)
// direction (string): Either "lend" or "loan".
func (api *API) NewOffer(currency string, amount, rate float64, period int, direction string) (offer Offer, err error) {
	currency = strings.ToUpper(currency)
	direction = strings.ToLower(direction)

	request := struct {
		URL       string  `json:"request"`
		Nonce     string  `json:"nonce"`
		Currency  string  `json:"currency"`
		Amount    float64 `json:"amount,string"`
		Rate      float64 `json:"rate,string"`
		Period    int     `json:"period"`
		Direction string  `json:"direction"`
	}{
		"/v1/offer/new",
		strconv.FormatInt(time.Now().UnixNano(), 10),
		currency,
		amount,
		rate,
		period,
		direction,
	}

	body, err := api.post(request.URL, request)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &offer)
	if err != nil || offer.ID == 0 { // Failed to unmarshal expected message
		// Attempt to unmarshal the error message
		errorMessage := ErrorMessage{}
		err = json.Unmarshal(body, &errorMessage)
		if err != nil { // Not expected message and not expected error, bailing...
			return
		}

		return offer, errors.New("API: " + errorMessage.Message)
	}

	return
}

///////////////////////////////////////
// API helper methods
///////////////////////////////////////

// CancelActiveOffers ...
func (api *API) CancelActiveOffers() (err error) {
	offers, err := api.ActiveOffers()
	if err != nil {
		return
	}

	for _, o := range offers {
		err = api.CancelOffer(o.ID)

		if err != nil {
			return
		}
	}

	return
}

// CancelActiveOffersByCurrency ...
func (api *API) CancelActiveOffersByCurrency(currency string) (err error) {
	currency = strings.ToLower(currency)

	offers, err := api.ActiveOffers()
	if err != nil {
		return
	}

	for _, o := range offers {
		if strings.ToLower(o.Currency) == currency {
			err = api.CancelOffer(o.ID)
			if err != nil {
				return
			}
		}
	}

	return
}

///////////////////////////////////////
// API query methods
///////////////////////////////////////

func (api *API) get(url string) (body []byte, err error) {
	resp, err := http.Get(APIURL + url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	return
}

func (api *API) post(url string, payload interface{}) (body []byte, err error) {
	// X-BFX-PAYLOAD
	// parameters-dictionary -> JSON encode -> base64
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return
	}
	payloadBase64 := base64.StdEncoding.EncodeToString(payloadJSON)

	// X-BFX-SIGNATURE
	// HMAC-SHA384(payload, api-secret) as hexadecimal
	h := hmac.New(sha512.New384, []byte(api.APISecret))
	h.Write([]byte(payloadBase64))
	signature := hex.EncodeToString(h.Sum(nil))

	// POST
	client := &http.Client{}
	req, err := http.NewRequest("POST", APIURL+url, bytes.NewBuffer(payloadJSON))
	if err != nil {
		return
	}

	req.Header.Add("X-BFX-APIKEY", api.APIKey)
	req.Header.Add("X-BFX-PAYLOAD", payloadBase64)
	req.Header.Add("X-BFX-SIGNATURE", signature)

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	return
}
