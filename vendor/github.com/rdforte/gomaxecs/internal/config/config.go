// Copyright 2004 Ryan Forte
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Package config provides the package configuration.
package config

import (
	"os"
	"strings"
	"time"
)

const (
	metaURIEnv  = "ECS_CONTAINER_METADATA_URI_V4"
	taskPath    = "/task"
	httpTimeout = 5
)

func New(opts ...Option) Config {
	uri := GetECSMetadataURI()

	cfg := Config{
		TaskMetadataURI:      uri + taskPath,
		ContainerMetadataURI: uri,
		Client: Client{
			HTTPTimeout:           time.Second * httpTimeout,
			DialTimeout:           time.Second,
			MaxIdleConns:          1,
			MaxIdleConnsPerHost:   1,
			DisableKeepAlives:     false, // keep connection alive for subsequent requests.
			IdleConnTimeout:       time.Second,
			TLSHandshakeTimeout:   time.Second,
			ResponseHeaderTimeout: time.Second,
		},
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}

// GetECSMetadataURI returns the ECS metadata URI.
func GetECSMetadataURI() string {
	uri := os.Getenv(metaURIEnv)
	return strings.TrimRight(uri, "/")
}

// Config represents the package configuration.
type Config struct {
	ContainerMetadataURI string
	TaskMetadataURI      string
	Client               Client
	log                  logger
}

type logger func(format string, args ...any)

// Client represents the HTTP client configuration.
type Client struct {
	HTTPTimeout           time.Duration
	DialTimeout           time.Duration
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	DisableKeepAlives     bool
	IdleConnTimeout       time.Duration
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
}

func (c Config) Log(format string, args ...any) {
	if c.log != nil {
		c.log(format, args...)
	}
}

// WithLogger sets the logger for the config.
func WithLogger(logger logger) Option {
	return func(cfg *Config) {
		cfg.log = logger
	}
}

// Option represents a configuration option for the config.
type Option func(*Config)
