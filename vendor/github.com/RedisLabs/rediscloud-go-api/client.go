package rediscloud_api

import (
	"log"
	"net/http"
	"os"

	"github.com/RedisLabs/rediscloud-go-api/internal"
	"github.com/RedisLabs/rediscloud-go-api/service/account"
	"github.com/RedisLabs/rediscloud-go-api/service/task"
)

type Client struct {
	Task    *task.Api
	Account *account.Api
}

func NewClient(configs ...Option) (*Client, error) {
	config := &Options{
		baseUrl:   "https://api.redislabs.com/v1",
		userAgent: userAgent,
		apiKey:    os.Getenv(ApiKeyEnvVar),
		secretKey: os.Getenv(SecretKeyEnvVar),
		logger:    log.New(os.Stderr, "", log.LstdFlags),
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

	t := task.NewApi(client, config.logger)
	a := account.NewApi(client)

	return &Client{
		Task:    t,
		Account: a,
	}, nil
}

type Options struct {
	baseUrl   string
	apiKey    string
	secretKey string
	userAgent string
	logger    Log
	transport http.RoundTripper
}

func (o Options) roundTripper() http.RoundTripper {
	return &credentialTripper{
		apiKey:    o.apiKey,
		secretKey: o.secretKey,
		wrapped:   o.transport,
	}
}

type Option func(*Options)

func Auth(apiKey string, secretKey string) Option {
	return func(options *Options) {
		options.apiKey = apiKey
		options.secretKey = secretKey
	}
}

func BaseUrl(url string) Option {
	return func(options *Options) {
		options.baseUrl = url
	}
}

func Transporter(transporter http.RoundTripper) Option {
	return func(options *Options) {
		options.transport = transporter
	}
}

func AdditionalUserAgent(additional string) Option {
	return func(options *Options) {
		options.userAgent += " " + additional
	}
}

func Logger(log Log) Option {
	return func(options *Options) {
		options.logger = log
	}
}

type Log interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

type credentialTripper struct {
	apiKey    string
	secretKey string
	wrapped   http.RoundTripper
}

func (c *credentialTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-Api-Key", c.apiKey)
	request.Header.Set("X-Api-Secret-Key", c.secretKey)

	return c.wrapped.RoundTrip(request)
}

var _ http.RoundTripper = &credentialTripper{}
