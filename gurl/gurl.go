package gurl

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type gurl struct {
	url    string
	data   interface{}
	body   BodyType
	param  map[string]string
	method string
	header map[string]string
	cookie map[string]string
	option Option
	client http.Client //custom client
}

type Option struct {
	Timeout    time.Duration // request timeout
	SkipVerify bool          // skip ssl verify
}

type Response *http.Response

type BodyType int

const (
	_ BodyType = iota
	TEXT
	FORM
	JSON
	XML
)

// New request
func New(method, url string) *gurl {
	return &gurl{url: url, method: method}
}

// Set options
func (this *gurl) Set(option Option) *gurl {
	this.option = option
	return this
}

// Set data and the type, default is JSON
func (this *gurl) Data(data interface{}, body ...BodyType) *gurl {
	this.data = data
	if len(body) > 0 {
		this.body = body[0]
	} else {
		this.body = JSON
	}
	return this
}

// Set params
func (this *gurl) Param(param map[string]string) *gurl {
	this.param = param
	return this
}

// Set headers
func (this *gurl) Header(header map[string]string) *gurl {
	this.header = header
	return this
}

// Set cookies
func (this *gurl) Cookie(cookie map[string]string) *gurl {
	this.cookie = cookie
	return this
}

// Set custom client
func (this *gurl) Client(client http.Client) *gurl {
	this.client = client
	return this
}

// Combined urls and parameters
func (this *gurl) urlWithParam() (err error) {
	if this.param == nil {
		return
	}

	var u *url.URL
	if u, err = url.Parse(this.url); err != nil {
		return
	}

	q := u.Query()
	for k, v := range this.param {
		q.Set(k, v)
	}

	u.RawQuery = q.Encode()
	this.url = u.String()

	return
}

func (this *gurl) Request() (response Response, err error) {
	if this.url == "" {
		return nil, errors.New("No url")
	} else {
		if err = this.urlWithParam(); err != nil {
			return
		}
	}
	if this.method == "" {
		return nil, errors.New("No method")
	} else {
		this.method = strings.ToUpper(this.method)
	}

	var payload io.Reader

	if this.data != nil && this.method != "GET" {
		switch this.body {
		case TEXT:
			payload = bytes.NewReader([]byte(this.data.(string)))
		case FORM:
			fdata := ""
			for k, v := range this.data.(map[string]string) {
				fdata += k + "=" + v + "&"
			}
			payload = strings.NewReader(fdata)
		case JSON:
			if jdata, err := json.Marshal(this.data); err != nil {
				return nil, err
			} else {
				payload = bytes.NewReader(jdata)
			}
		case XML:
			if xdata, err := xml.Marshal(this.data); err != nil {
				return nil, err
			} else {
				payload = bytes.NewReader(xdata)
			}
		}
	}

	request, err := http.NewRequest(this.method, this.url, payload)
	if err != nil {
		return
	}

	switch this.body {
	case FORM:
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	case JSON:
		request.Header.Set("Content-Type", "application/json")
	}

	if this.header != nil {
		for k, v := range this.header {
			request.Header.Set(k, v)
		}
	}

	// options
	opt := this.option

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: opt.SkipVerify},
	}

	this.client.Transport = tr
	this.client.Timeout = opt.Timeout

	response, err = this.client.Do(request)
	return
}

func (this *gurl) Do() (body []byte, err error) {
	var resp Response
	if resp, err = this.Request(); err != nil {
		return
	}
	body, err = ioutil.ReadAll(resp.Body)
	return
}
