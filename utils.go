package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"regexp"
	"time"

	"github.com/fatih/color"
)

func NewClient() *http.Client {
	Jar, _ := cookiejar.New(nil)

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			MaxVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				0x0a0a, 0x1301, 0x1302, 0x1303, 0xc02b, 0xc02f, 0xc02c, 0xc030,
				0xcca9, 0xcca8, 0xc013, 0xc014, 0x009c, 0x009d, 0x002f, 0x0035,
			},
			CurvePreferences: []tls.CurveID{
				tls.CurveID(0x0a0a),
				tls.X25519,
				tls.CurveP256,
				tls.CurveP384,
			},
			InsecureSkipVerify: true,
		},
	}

	c := &http.Client{
		Jar:       Jar,
		Transport: t,
	}

	return c
}

func LoadProfile(f string) profileSt {
	fp, err := os.Open(f)

	if err != nil {
		log.Fatal("Error opening profile")
	}

	defer fp.Close()

	fb, _ := ioutil.ReadAll(fp)

	var loaded profileSt

	json.Unmarshal(fb, &loaded)

	return loaded
}

func ReFindall(regex string, s string) []string {
	r := regexp.MustCompile(regex)
	tokens := r.FindAllStringSubmatch(s, -1)
	out := make([]string, len(tokens))
	for i := range out {
		out[i] = tokens[i][1]
	}
	return out
}

func tstamp() {
	ts := time.Now().Format("2006-01-02@15:04:05")
	fmt.Printf("[%s] ", ts)

}

func Emit(status string, c string) {
	switch c {
	case "b":
		tstamp()
		color.Blue("%s", status)
	case "c":
		tstamp()
		color.Cyan("%s", status)
	case "g":
		tstamp()
		color.Green("%s", status)
	case "r":
		tstamp()
		color.Red("%s", status)
	case "y":
		tstamp()
		color.Yellow("%s", status)
	default:
		tstamp()
		color.White("%s", status)
	}
}
