package bsonrpc

import (
	"errors"
	"fmt"
	"github.com/skynetservices/mgo/bson"
	"io"
)

type Encoder struct {
	w io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

func (e *Encoder) Encode(v interface{}) (err error) {
	buf, err := bson.Marshal(v)
	if err != nil {
		return
	}
	_, err = e.w.Write(buf)

	return
}

type Decoder struct {
	r io.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

func (d *Decoder) Decode(pv interface{}) (err error) {
	var lbuf [4]byte
	n, err := d.r.Read(lbuf[:])
	if n == 0 {
		err = io.EOF
		return
	}
	if n != 4 {
		err = errors.New(fmt.Sprintf("Corrupted BSON stream: could only read %d", n))
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
	_, err = d.r.Read(buf[4:])
	if err != nil {
		return
	}

	err = bson.Unmarshal(buf, pv)

	return
}
