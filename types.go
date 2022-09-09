package main

import (
	"net/http"
)

type profileSt struct {
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	CardHolder string `json:"cardHolder"`
	CardNum    string `json:"cardNum"`
	Cvv        string `json:"cvv"`
	Expmonth   string `json:"expmonth"`
	Expyear    string `json:"expyear"`
	Billing    struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Address1  string `json:"address1"`
		Address2  string `json:"address2"`
		State     struct {
			Short string `json:"short"`
			Long  string `json:"long"`
		} `json:"state"`
		City    string `json:"city"`
		Country struct {
			Short string `json:"short"`
			Long  string `json:"long"`
		} `json:"country"`
		Zip string `json:"zip"`
	} `json:"billing"`
}

type task struct {
	Product string
	Profile profileSt
	Delays  [2]int
	Client  http.Client
	State   string
}
