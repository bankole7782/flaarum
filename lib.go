// this package 'flaarum' is the golang library for communicating with the flaarum server.
package flaarum

import (
	"net/http"
	"crypto/tls"
	"github.com/pkg/errors"
	"strings"
	"io/ioutil"
	"github.com/bankole7782/flaarum/flaarum_shared"
	"time"
	"net/url"
)


var httpCl *http.Client

func init() {
	config := &tls.Config { InsecureSkipVerify: true}
	tr := &http.Transport{TLSClientConfig: config}

	httpCl = &http.Client{Transport: tr}
}

type Client struct {
	Addr string
	KeyStr string
	ProjName string
}

func NewClient(addr, keyStr, projName string) Client {
	if ! strings.HasSuffix(addr, "/") {
		addr += "/"
	}

	return Client{addr, keyStr, projName}
}


func (cl *Client) Ping() error {
  urlValues := url.Values{}
  urlValues.Set("key-str", cl.KeyStr)

	resp, err := httpCl.PostForm(cl.Addr + "is-flaarum", urlValues)
	if err != nil {
		return errors.Wrap(err, "http error")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if resp.StatusCode == 200 {
		if string(body) == "yeah-flaarum" {
			return nil
		} else {
			return errors.New("Unexpected Error in confirming that the server is a flaarum store.")
		}
	} else {
		return errors.New(string(body))
	}
}

// Converts a time.Time to the date format expected in flaarum
func RightDateFormat(d time.Time) string {
	return d.Format(flaarum_shared.BROWSER_DATE_FORMAT)
}

// Converts a time.Time to the datetime format expected in flaarum
func RightDateTimeFormat(d time.Time) string {
	return d.Format(flaarum_shared.BROWSER_DATETIME_FORMAT)
}
