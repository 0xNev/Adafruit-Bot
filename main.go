package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func start() {

	//Create new task
	t := &task{
		Product: "5584",
		Profile: LoadProfile("profile.json"),
		Delays:  [2]int{4500, 6500}, //retry and error delay respectivly
		Client:  *NewClient(),
		State:   "monitor",
	}

	var err error

	var securityToken string
	var csrf_token string

	for t.State != "kill" {
		switch t.State {
		case "monitor":
			inStock, err := MonitorProduct(t)

			if err != nil {
				Emit(err.Error(), "r")
				time.Sleep(time.Duration(t.Delays[1]) * time.Millisecond)
			} else if inStock {
				t.State = "getSecToken"
			} else {
				time.Sleep(time.Duration(t.Delays[0]) * time.Millisecond)
			}

		case "getSecToken":
			securityToken, err = GetProductPage(t)

			if err != nil {
				Emit(err.Error(), "r")
				time.Sleep(time.Duration(t.Delays[1]) * time.Millisecond)
			} else {
				t.State = "cart"
			}
		case "cart":
			err = CartProduct(t, securityToken)

			if err != nil {
				Emit(err.Error(), "r")
				time.Sleep(time.Duration(t.Delays[1]) * time.Millisecond)
			} else {
				t.State = "getCsrf"
			}
		case "getCsrf":
			csrf_token, err = GetCsrf(t)

			if err != nil {
				Emit(err.Error(), "r")
				time.Sleep(time.Duration(t.Delays[1]) * time.Millisecond)
			} else {
				t.State = "checkout"
			}
		case "checkout":
			err = SubmitBilling(t, csrf_token)

			if err != nil {
				Emit(err.Error(), "r")
				time.Sleep(time.Duration(t.Delays[1]) * time.Millisecond)
			} else {
				t.State = "finalize"
			}
		case "finalize":
			err = FinalizeOrder(t, csrf_token)

			if err != nil {
				Emit(err.Error(), "r")
				time.Sleep(time.Duration(t.Delays[1]) * time.Millisecond)
			} else {
				t.State = "kill"
			}

		}

	}

}

func MonitorProduct(t *task) (bool, error) {
	Emit("Monitoring product...", "d")

	req, _ := http.NewRequest("GET", "https://www.adafruit.com/product/"+t.Product, nil)

	for k, v := range map[string]string{
		`authority`:                 `www.adafruit.com`,
		`accept`:                    `text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9`,
		`accept-language`:           `en-US,en;q=0.9`,
		`cache-control`:             `max-age=0`,
		`dnt`:                       `1`,
		`sec-ch-ua`:                 `"Chromium";v="104", " Not A;Brand";v="99", "Google Chrome";v="104"`,
		`sec-ch-ua-mobile`:          `?0`,
		`sec-ch-ua-platform`:        `"Windows"`,
		`sec-fetch-dest`:            `document`,
		`sec-fetch-mode`:            `navigate`,
		`sec-fetch-site`:            `none`,
		`sec-fetch-user`:            `?1`,
		`upgrade-insecure-requests`: `1`,
		`user-agent`:                `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36`,
	} {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil || resp.StatusCode > 201 {
		return false, errors.New("Error monitoring product")
	}

	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	if len(ReFindall(`itemprop="availability".*>(.+)`, string(body))) > 0 {
		if ReFindall(`itemprop="availability".*>(.+)<`, string(body))[0] == "In stock" {
			return true, nil
		} else if ReFindall(`itemprop="availability".*>(.+)<`, string(body))[0] == "Out of stock" {
			return false, nil
		}

	}

	return false, errors.New("Error monitoring product")

}

func GetProductPage(t *task) (string, error) {
	Emit("Fetching product page", "d")

	req, _ := http.NewRequest("GET", "https://www.adafruit.com/product/"+t.Product, nil)

	for k, v := range map[string]string{
		`authority`:                 `www.adafruit.com`,
		`accept`:                    `text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9`,
		`accept-language`:           `en-US,en;q=0.9`,
		`cache-control`:             `max-age=0`,
		`dnt`:                       `1`,
		`sec-ch-ua`:                 `"Chromium";v="104", " Not A;Brand";v="99", "Google Chrome";v="104"`,
		`sec-ch-ua-mobile`:          `?0`,
		`sec-ch-ua-platform`:        `"Windows"`,
		`sec-fetch-dest`:            `document`,
		`sec-fetch-mode`:            `navigate`,
		`sec-fetch-site`:            `none`,
		`sec-fetch-user`:            `?1`,
		`upgrade-insecure-requests`: `1`,
		`user-agent`:                `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36`,
	} {
		req.Header.Set(k, v)
	}

	resp, err := t.Client.Do(req)

	if err != nil || resp.StatusCode > 201 {
		return "", errors.New("Bad response")
	}

	defer resp.Body.Close()

	for _, cookie := range resp.Cookies() {
		t.Client.Jar.SetCookies(req.URL, []*http.Cookie{cookie})
	}

	respBody, _ := ioutil.ReadAll(resp.Body)

	if len(ReFindall(`"securityToken" value="(.*?)"`, string(respBody))) > 0 {
		return ReFindall(`"securityToken" value="(.*?)"`, string(respBody))[0], nil
	}

	return "", errors.New("Failed to get security token")
}

func CartProduct(t *task, securityToken string) error {
	Emit("Attempting ATC", "y")

	params := url.Values{}

	for k, v := range map[string]string{
		`securityToken`: securityToken,
		`action`:        `add_product`,
		`pid`:           t.Product,
		`qty`:           `1`,
		`source_page`:   `product`,
		`source_id`:     t.Product,
	} {
		params.Add(k, v)
	}

	body := strings.NewReader(params.Encode())

	req, _ := http.NewRequest("POST", "https://www.adafruit.com/added", body)

	for k, v := range map[string]string{
		`authority`:                 `www.adafruit.com`,
		`accept`:                    `text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9`,
		`accept-language`:           `en-US,en;q=0.9`,
		`cache-control`:             `max-age=0`,
		`dnt`:                       `1`,
		`content-type`:              `application/x-www-form-urlencoded`,
		`origin`:                    `https://www.adafruit.com`,
		`referer`:                   `https://www.adafruit.com/product/` + t.Product,
		`sec-ch-ua`:                 `"Chromium";v="104", " Not A;Brand";v="99", "Google Chrome";v="104"`,
		`sec-ch-ua-mobile`:          `?0`,
		`sec-ch-ua-platform`:        `"Windows"`,
		`sec-fetch-dest`:            `document`,
		`sec-fetch-mode`:            `navigate`,
		`sec-fetch-site`:            `same-origin`,
		`sec-fetch-user`:            `?1`,
		`upgrade-insecure-requests`: `1`,
		`user-agent`:                `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36`,
	} {
		req.Header.Add(k, v)
	}

	resp, err := t.Client.Do(req)

	if err != nil || resp.StatusCode > 201 {
		return errors.New("Bad response")
	}

	defer resp.Body.Close()

	for _, cookie := range resp.Cookies() {
		t.Client.Jar.SetCookies(req.URL, []*http.Cookie{cookie})
	}

	Emit("Added item to cart", "b")

	return nil

}

func GetCsrf(t *task) (string, error) {

	Emit("Scraping CSRF token", "c")

	var csrf_token string

	req, _ := http.NewRequest("GET", "https://www.adafruit.com/checkout", nil)

	for k, v := range map[string]string{
		`authority`:                 `www.adafruit.com`,
		`accept`:                    `text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9`,
		`accept-language`:           `en-US,en;q=0.9`,
		`referer`:                   `https://www.adafruit.com/shopping_cart`,
		`sec-ch-ua`:                 `"Chromium";v="104", " Not A;Brand";v="99", "Google Chrome";v="104"`,
		`sec-ch-ua-mobile`:          `?0`,
		`sec-ch-ua-platform`:        `"Windows"`,
		`sec-fetch-dest`:            `document`,
		`sec-fetch-mode`:            `navigate`,
		`sec-fetch-site`:            `same-origin`,
		`sec-fetch-user`:            `?1`,
		`upgrade-insecure-requests`: `1`,
		`user-agent`:                `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36`,
	} {
		req.Header.Set(k, v)
	}

	resp, err := t.Client.Do(req)

	if err != nil || resp.StatusCode > 201 {
		return "", errors.New("Bad response")
	}

	if resp.Request.URL.String() != "https://www.adafruit.com/checkout?step=1" {
		return "", errors.New("Could not find redirect (Checkout)")
	}

	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)

	if len(ReFindall(`csrf_token" value="(.*)"`, string(respBody))) > 0 {
		csrf_token = ReFindall(`csrf_token" value="(.*)"`, string(respBody))[0]
	} else {
		return "", errors.New("Failed to get CSRF")
	}

	return csrf_token, nil
}

func SubmitBilling(t *task, csrf_token string) error {

	Emit("Submitting billing details", "y")

	headers := map[string]string{
		`authority`:                 `www.adafruit.com`,
		`accept`:                    `text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9`,
		`accept-language`:           `en-US,en;q=0.9`,
		`cache-control`:             `max-age=0`,
		`content-type`:              `application/x-www-form-urlencoded`,
		`origin`:                    `https://www.adafruit.com`,
		`referer`:                   `https://www.adafruit.com/checkout?step=1`,
		`sec-ch-ua`:                 `"Chromium";v="104", " Not A;Brand";v="99", "Google Chrome";v="104"`,
		`sec-ch-ua-mobile`:          `?0`,
		`sec-ch-ua-platform`:        `"Windows"`,
		`sec-fetch-dest`:            `document`,
		`sec-fetch-mode`:            `navigate`,
		`sec-fetch-site`:            `same-origin`,
		`sec-fetch-user`:            `?1`,
		`upgrade-insecure-requests`: `1`,
		`user-agent`:                `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36`,
	}

	//Step 1

	params := url.Values{}

	for k, v := range map[string]string{
		`csrf_token`:    csrf_token,
		`email_address`: t.Profile.Email,
		`action`:        `save_one`,
	} {
		params.Add(k, v)
	}

	body := strings.NewReader(params.Encode())

	req, _ := http.NewRequest("POST", "https://www.adafruit.com/checkout", body)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := t.Client.Do(req)

	if err != nil || resp.StatusCode > 201 {
		return errors.New("Bad response")
	}

	if resp.Request.URL.String() != "https://www.adafruit.com/checkout?step=2" {
		return errors.New("Failed to enter billing(1)")
	}

	defer resp.Body.Close()

	respBody, _ := ioutil.ReadAll(resp.Body)

	var stateCode string

	r := fmt.Sprintf(`value="(\d{2})">%s<\/option>`, t.Profile.Billing.State.Long)

	if len(ReFindall(r, string(respBody))) > 0 {
		stateCode = string(ReFindall(r, string(respBody))[0])
	}

	//Step 2

	params = url.Values{}

	for k, v := range map[string]string{
		`csrf_token`:          csrf_token,
		`delivery_use_anyway`: `0`,
		`delivery_name`:       t.Profile.Billing.FirstName + " " + t.Profile.Billing.LastName,
		`delivery_company`:    ``,
		`delivery_address1`:   t.Profile.Billing.Address1,
		`delivery_address2`:   t.Profile.Billing.Address2,
		`delivery_city`:       t.Profile.Billing.City,
		`delivery_state`:      stateCode,
		`delivery_postcode`:   t.Profile.Billing.Zip,
		`delivery_country`:    `223`,
		`delivery_phone`:      t.Profile.Phone,
		`billing_use_anyway`:  `0`,
		`billing_name`:        ``,
		`billing_company`:     ``,
		`billing_address1`:    ``,
		`billing_address2`:    ``,
		`billing_city`:        ``,
		`billing_state`:       `0`,
		`billing_postcode`:    ``,
		`billing_country`:     `223`,
		`billing_phone`:       ``,
		`action`:              `save_two`,
	} {
		params.Add(k, v)
	}

	body = strings.NewReader(params.Encode())

	req, _ = http.NewRequest("POST", "https://www.adafruit.com/checkout", body)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err = t.Client.Do(req)

	if err != nil || resp.StatusCode > 201 {
		return errors.New("Bad response")
	}

	if resp.Request.URL.String() != "https://www.adafruit.com/checkout?step=3" {
		return errors.New("Failed to enter billing(2)")
	}

	defer resp.Body.Close()

	//Step 3 Shipping rate

	params = url.Values{}

	for k, v := range map[string]string{
		`csrf_token`: csrf_token,
		`shipping`:   "usps_First-Class Package Service - Retail&lt;sup&gt;&#8482;&lt;/sup&gt;",
		`action`:     `save_three`,
	} {
		params.Add(k, v)
	}

	body = strings.NewReader(params.Encode())

	req, _ = http.NewRequest("POST", "https://www.adafruit.com/checkout", body)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err = t.Client.Do(req)

	if err != nil || resp.StatusCode > 201 {
		return errors.New("Bad response")
	}

	if resp.Request.URL.String() != "https://www.adafruit.com/checkout?step=4" {
		return errors.New("Failed to enter billing(3)")
	}

	defer resp.Body.Close()

	//Step 4

	params = url.Values{}

	for k, v := range map[string]string{
		`csrf_token`:                        csrf_token,
		`action`:                            `save_four`,
		`payment`:                           `authorizenet_aim`,
		`authorizenet_aim_cc_owner`:         t.Profile.CardHolder,
		`authorizenet_aim_cc_number`:        t.Profile.CardNum,
		`authorizenet_aim_cc_expires_month`: t.Profile.Expmonth,
		`authorizenet_aim_cc_expires_year`:  t.Profile.Expyear,
		`authorizenet_aim_cc_cvv`:           ``,
		`card-type`:                         `amex`,
		`po_payment_type`:                   `replacement`,
	} {
		params.Add(k, v)
	}

	body = strings.NewReader(params.Encode())

	req, _ = http.NewRequest("POST", "https://www.adafruit.com/checkout", body)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err = t.Client.Do(req)

	if err != nil || resp.StatusCode > 201 {
		return errors.New("Bad response")
	}

	defer resp.Body.Close()

	respBody, _ = ioutil.ReadAll(resp.Body)

	if resp.Request.URL.String() != "https://www.adafruit.com/checkout" {
		return errors.New("Failed to enter billing(4)")
	}

	return nil

}

func FinalizeOrder(t *task, csrf_token string) error {
	Emit("Finalizing Order", "b")
	params := url.Values{}

	for k, v := range map[string]string{
		`csrf_token`: csrf_token,
		`cc_owner`:   t.Profile.CardHolder,
		`cc_expires`: t.Profile.Expmonth + t.Profile.Expyear,
		`cc_type`:    `Visa`,
		`cc_number`:  strings.ReplaceAll(t.Profile.CardNum, " ", ""),
		`cc_cvv`:     ``,
		`zenid`:      ``,
	} {
		params.Add(k, v)
	}

	body := strings.NewReader(params.Encode())

	req, _ := http.NewRequest("POST", "https://www.adafruit.com/index.php?main_page=checkout_process", body)

	for k, v := range map[string]string{
		`authority`:                 `www.adafruit.com`,
		`accept`:                    `text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9`,
		`accept-language`:           `en-US,en;q=0.9`,
		`cache-control`:             `max-age=0`,
		`content-type`:              `application/x-www-form-urlencoded`,
		`origin`:                    `https://www.adafruit.com`,
		`referer`:                   `https://www.adafruit.com/checkout?step=4`,
		`sec-ch-ua`:                 `"Chromium";v="104", " Not A;Brand";v="99", "Google Chrome";v="104"`,
		`sec-ch-ua-mobile`:          `?0`,
		`sec-ch-ua-platform`:        `"Windows"`,
		`sec-fetch-dest`:            `document`,
		`sec-fetch-mode`:            `navigate`,
		`sec-fetch-site`:            `same-origin`,
		`sec-fetch-user`:            `?1`,
		`upgrade-insecure-requests`: `1`,
		`user-agent`:                `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36`,
	} {
		req.Header.Set(k, v)
	}

	resp, err := t.Client.Do(req)

	if err != nil || resp.StatusCode > 201 {
		return errors.New("Bad response")
	}

	defer resp.Body.Close()

	respBody, _ := ioutil.ReadAll(resp.Body)

	fmt.Println(string(respBody))

	Emit("Check email", "g")

	return nil
}

func start() {
	instance()

}
