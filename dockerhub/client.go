package dockerhub

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/jacexh/requests"
)

// Client docker registry v2 api的实现
type Client struct {
	Host   string
	Scheme string
	option Option
	client *requests.Session
}

func parseURL(url string) (string, string, error) {
	reg, err := regexp.Compile("^(?P<Scheme>http[s]?)://(?P<Host>[\\w.-]*)[/]?$")
	if err != nil {
		return "", "", err
	}
	subs := reg.FindStringSubmatch(url)
	if len(subs) != 3 {
		return "", "", errors.New("bad url")
	}
	return subs[1], subs[2], nil
}

func NewClient(opt Option) (*Client, error) {
	client := &Client{option: opt}

	if opt.URL == "" {
		client.Host = ""
		client.Scheme = "https"
	} else {
		scheme, host, err := parseURL(opt.URL)
		if err != nil {
			return nil, err
		}
		client.Scheme = scheme
		client.Host = host
	}

	var session *requests.Session
	var auth = opt.AuthToken
	if auth == "" {
		auth = base64.StdEncoding.EncodeToString([]byte(opt.Username + ":" + opt.Password))
	}
	session = requests.NewSession(requests.Option{Headers: requests.Any{"Authorization": fmt.Sprintf("Basic " + auth)}})
	client.client = session
	return client, nil
}

func (client *Client) url(path string, v ...interface{}) string {
	return client.Scheme + "://" + client.Host + fmt.Sprintf(path, v...)
}

// ListTags listing image tags, pagination not work: https://github.com/docker/distribution/issues/1936
func (client *Client) ListTags(ctx context.Context, name string, opt *ListTagsOption) (*ResponseTag, *http.Response, error) {
	res := new(Response)
	var params requests.Params
	if opt != nil {
		params.Query = map[string]string{"n": strconv.Itoa(opt.Number)}
	}

	raw, _, err := client.client.GetWithContext(
		ctx,
		client.url("/v2/%s/tags/list", name),
		params,
		requests.UnmarshalJSONResponse(res))
	if err != nil {
		return nil, nil, err
	}
	if res.Error() != nil {
		return nil, raw, res.Error()
	}
	return res.ResponseTag, raw, nil
}

// GetManifest fetch the manifest identified by name and reference where reference can be a tag or digest
func (client *Client) GetManifest(ctx context.Context, name, reference string) (*ResponseManifest, *http.Response, error) {
	res := new(Response)
	raw, _, err := client.client.GetWithContext(ctx,
		client.url("/v2/%s/manifests/%s", name, reference),
		requests.Params{},
		requests.UnmarshalJSONResponse(res),
	)
	if err != nil {
		return nil, nil, err
	}
	if res.Error() != nil {
		return nil, raw, res.Error()
	}
	return res.ResponseManifest, raw, nil
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Response 判断response是否异常，如果有，返回第一个error
func (res *Response) Error() error {
	if len(res.Errors) > 0 {
		return res.Errors[0]
	}
	return nil
}
