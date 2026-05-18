// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package snappy registers a gRPC snappy compressor compatible with the
// collector's configgrpc package.
package snappy // import "go.opentelemetry.io/collector/internal/grpccompression/snappy"

import (
	"io"
	"sync"

	snappylib "github.com/golang/snappy"
	"google.golang.org/grpc/encoding"
)

var snappyNewBufferedWriter = snappylib.NewBufferedWriter

var snappyWriterReset = func(w *snappylib.Writer, dst io.Writer) {
	w.Reset(dst)
}

var snappyNewReader = snappylib.NewReader

var snappyReaderReset = func(r *snappylib.Reader, src io.Reader) {
	r.Reset(src)
}

// Name is the content-coding used for snappy-compressed gRPC payloads.
const Name = "snappy"

func init() {
	registerCompressor(encoding.GetCompressor, encoding.RegisterCompressor)
}

func registerCompressor(get func(string) encoding.Compressor, register func(encoding.Compressor)) bool {
	if get(Name) != nil {
		return false
	}

	register(newCompressor())
	return true
}

type compressor struct {
	poolCompressor   sync.Pool
	poolDecompressor sync.Pool
}

func newCompressor() *compressor {
	c := &compressor{}
	c.poolCompressor.New = func() any {
		return snappyNewBufferedWriter(io.Discard)
	}
	return c
}

func (c *compressor) Compress(w io.Writer) (io.WriteCloser, error) {
	z, ok := c.poolCompressor.Get().(*snappylib.Writer)
	if !ok {
		z = snappyNewBufferedWriter(w)
	} else {
		snappyWriterReset(z, w)
	}

	return &writer{
		Writer: z,
		pool:   &c.poolCompressor,
	}, nil
}

func (c *compressor) Decompress(r io.Reader) (io.Reader, error) {
	z, ok := c.poolDecompressor.Get().(*snappylib.Reader)
	if !ok {
		return &reader{
			Reader: snappyNewReader(r),
			pool:   &c.poolDecompressor,
		}, nil
	}

	snappyReaderReset(z, r)
	return &reader{
		Reader: z,
		pool:   &c.poolDecompressor,
	}, nil
}

func (c *compressor) Name() string {
	return Name
}

type writer struct {
	*snappylib.Writer
	pool *sync.Pool
	once sync.Once
}

func (z *writer) Close() error {
	err := z.Writer.Close()
	z.release()
	return err
}

func (z *writer) release() {
	z.once.Do(func() {
		z.pool.Put(z.Writer)
	})
}

type reader struct {
	*snappylib.Reader
	pool *sync.Pool
	once sync.Once
}

func (z *reader) Read(p []byte) (int, error) {
	n, err := z.Reader.Read(p)
	if err != nil {
		z.release()
	}
	return n, err
}

func (z *reader) release() {
	z.once.Do(func() {
		z.pool.Put(z.Reader)
	})
}
