package bson

import (
	"fmt"
	"io"
	"net/rpc"

	"github.com/micro/go-bson"
)

type bsonClientCodec struct {
	conn    io.ReadWriteCloser
	Encoder *bsonEncoder
	Decoder *bsonDecoder
}

// NewBsonClientCodec creates a bson ClientCodec
func NewBsonClientCodec(conn io.ReadWriteCloser) rpc.ClientCodec {
	return &bsonClientCodec{
		conn:    conn,
		Encoder: newBsonEncoder(conn),
		Decoder: newBsonDecoder(conn),
	}
}

func (cc *bsonClientCodec) WriteRequest(req *rpc.Request, v interface{}) (err error) {
	err = cc.Encoder.Encode(req)
	if err != nil {
		cc.Close()
		return
	}

	err = cc.Encoder.Encode(v)
	if err != nil {
		return
	}

	return
}

func (cc *bsonClientCodec) ReadResponseHeader(res *rpc.Response) (err error) {
	err = cc.Decoder.Decode(res)
	return
}

func (cc *bsonClientCodec) ReadResponseBody(v interface{}) (err error) {
	err = cc.Decoder.Decode(v)
	return
}

func (cc *bsonClientCodec) Close() (err error) {
	err = cc.conn.Close()
	return
}

type bsonServerCodec struct {
	conn    io.ReadWriteCloser
	Encoder *bsonEncoder
	Decoder *bsonDecoder
}

// NewBsonServerCodec creates a bson ServerCodec
func NewBsonServerCodec(conn io.ReadWriteCloser) rpc.ServerCodec {
	return &bsonServerCodec{
		conn:    conn,
		Encoder: newBsonEncoder(conn),
		Decoder: newBsonDecoder(conn),
	}
}

func (sc *bsonServerCodec) ReadRequestHeader(rq *rpc.Request) (err error) {
	err = sc.Decoder.Decode(rq)
	return
}

func (sc *bsonServerCodec) ReadRequestBody(v interface{}) (err error) {
	err = sc.Decoder.Decode(v)
	return
}

func (sc *bsonServerCodec) WriteResponse(rs *rpc.Response, v interface{}) (err error) {
	err = sc.Encoder.Encode(rs)
	if err != nil {
		return
	}
	err = sc.Encoder.Encode(v)
	if err != nil {
		return
	}
	return
}

func (sc *bsonServerCodec) Close() (err error) {
	err = sc.conn.Close()
	return
}

type bsonEncoder struct {
	w io.Writer
}

func newBsonEncoder(w io.Writer) *bsonEncoder {
	return &bsonEncoder{w: w}
}

func (e *bsonEncoder) Encode(v interface{}) (err error) {
	buf, err := bson.Marshal(v)
	if err != nil {
		return
	}

	n, err := e.w.Write(buf)
	if err != nil {
		return
	}

	if l := len(buf); n != l {
		err = fmt.Errorf("Wrote %d bytes, should have wrote %d", n, l)
	}

	return
}

type bsonDecoder struct {
	r io.Reader
}

func newBsonDecoder(r io.Reader) *bsonDecoder {
	return &bsonDecoder{r: r}
}

func (d *bsonDecoder) Decode(pv interface{}) (err error) {
	var lbuf [4]byte
	n, err := d.r.Read(lbuf[:])

	if n != 4 {
		err = fmt.Errorf("Corrupted BSON stream: could only read %d", n)
		return
	}
	if err != nil {
		return
	}

	length := (int(lbuf[0]) << 0) |
		(int(lbuf[1]) << 8) |
		(int(lbuf[2]) << 16) |
		(int(lbuf[3]) << 24)

	buf := make([]byte, length)
	copy(buf[0:4], lbuf[:])

	n, err = io.ReadFull(d.r, buf[4:])
	if err != nil {
		return
	}

	if n+4 != length {
		err = fmt.Errorf("Expected %d bytes, read %d", length, n)
		return
	}
	err = bson.Unmarshal(buf, pv)

	return
}
