package rediscloud_api

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/RedisLabs/rediscloud-go-api/internal"
	"github.com/RedisLabs/rediscloud-go-api/service/account"
	"github.com/RedisLabs/rediscloud-go-api/service/cloud_accounts"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
)

type Client struct {
	Account      *account.API
	CloudAccount *cloud_accounts.API
	Database     *databases.API
	Subscription *subscriptions.API
}

func NewClient(configs ...Option) (*Client, error) {
	config := &Options{
		baseUrl:   "https://api.redislabs.com/v1",
		userAgent: userAgent,
		apiKey:    os.Getenv(AccessKeyEnvVar),
		secretKey: os.Getenv(SecretKeyEnvVar),
		logger:    &defaultLogger{},
		transport: http.DefaultTransport,
	}

	for _, option := range configs {
		option(config)
	}

	httpClient := &http.Client{
		Transport: config.roundTripper(),
	}

	client, err := internal.NewHttpClient(httpClient, config.baseUrl)
	if err != nil {
		return nil, err
	}

	t := internal.NewAPI(client, config.logger)

	a := account.NewAPI(client)
	c := cloud_accounts.NewAPI(client, t, config.logger)
	d := databases.NewAPI(client, t, config.logger)
	s := subscriptions.NewAPI(client, t, config.logger)

	return &Client{
		Account:      a,
		CloudAccount: c,
		Database:     d,
		Subscription: s,
	}, nil
}

type Options struct {
	baseUrl     string
	apiKey      string
	secretKey   string
	userAgent   string
	logger      Log
	transport   http.RoundTripper
	logRequests bool
}

func (o Options) roundTripper() http.RoundTripper {
	return &credentialTripper{
		apiKey:      o.apiKey,
		secretKey:   o.secretKey,
		wrapped:     o.transport,
		logRequests: o.logRequests,
		logger:      o.logger,
		userAgent:   o.userAgent,
	}
}

type Option func(*Options)

// Auth is used to set the authentication credentials - will otherwise default to using environment variables
// for the credentials.
func Auth(apiKey string, secretKey string) Option {
	return func(options *Options) {
		options.apiKey = apiKey
		options.secretKey = secretKey
	}
}

// BaseURL sets the URL to use for the API endpoint - will default to `https://api.redislabs.com/v1`.
func BaseURL(url string) Option {
	return func(options *Options) {
		options.baseUrl = url
	}
}

// LogRequests allows the logging of HTTP request and responses - will default to false (disabled).
func LogRequests(enable bool) Option {
	return func(options *Options) {
		options.logRequests = enable
	}
}

// Transporter allows the customisation of the RoundTripper used to communicate with the API - will default to the
// Go default.
func Transporter(transporter http.RoundTripper) Option {
	return func(options *Options) {
		options.transport = transporter
	}
}

// AdditionalUserAgent allows extra information to be appended to the user agent sent in all requests to the API.
func AdditionalUserAgent(additional string) Option {
	return func(options *Options) {
		options.userAgent += " " + additional
	}
}

// Logger allows for a custom implementation to handle the debug log messages - defaults to using the Go standard log
// package.
func Logger(log Log) Option {
	return func(options *Options) {
		options.logger = log
	}
}

type Log interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

type defaultLogger struct{}

func (d *defaultLogger) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (d *defaultLogger) Println(v ...interface{}) {
	log.Println(v...)
}

type credentialTripper struct {
	apiKey      string
	secretKey   string
	wrapped     http.RoundTripper
	logRequests bool
	logger      Log
	userAgent   string
}

func (c *credentialTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", c.userAgent)

	if c.logRequests {
		data, _ := httputil.DumpRequestOut(request, true)
		if data != nil {
			c.logger.Printf(`DEBUG: Request %s:
---[ REQUEST ]---
%s`, request.URL.Path, prettyPrint(data))
		}
	}

	// Credentials added _after_ the request was logged to avoid accidentally logging them
	request.Header.Set("X-Api-Key", c.apiKey)
	request.Header.Set("X-Api-Secret-Key", c.secretKey)

	response, err := c.wrapped.RoundTrip(request)
	if err != nil {
		return response, err
	}

	if c.logRequests {
		data, _ := httputil.DumpResponse(response, true)
		if data != nil {
			c.logger.Printf(`DEBUG: Response %s:
---[ RESPONSE ]---
%s`, request.URL.Path, prettyPrint(data))
		}
	}
	return response, nil
}

func prettyPrint(data []byte) string {
	lines := strings.Split(string(data), "\n")
	// A JSON body that wasn't indented would have ended up as a single line in the dumped information,
	// so try and find a line which is valid JSON and then indent it
	for i, line := range lines {
		asBytes := []byte(line)
		if json.Valid(asBytes) {
			var indented bytes.Buffer
			if err := json.Indent(&indented, asBytes, "", "  "); err == nil {
				lines[i] = indented.String()
			}
		}
	}
	return strings.Join(lines, "\n")
}

var _ http.RoundTripper = &credentialTripper{}
