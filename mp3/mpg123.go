// Package mp3 binds to the libmpg123 library.
package mp3

/*
#include <mpg123.h>
#cgo LDFLAGS: -lmpg123
*/
import "C"

import (
	"errors"
	"fmt"
	"io"
	"log"
	"unsafe"
)

var initError error

// ReadBufferSize is the size of the buffer passed to io.Readers passed to NewDecoder.
var ReadBufferSize int = 4096

func init() {
	initError = toError(C.mpg123_init())
}

// toError returns an error from the mpg123 integer numeral.
func toError(e C.int) error {
	if e == C.MPG123_OK {
		return nil
	}

	s := C.mpg123_plain_strerror(e)
	err := errors.New(C.GoString(s))
	// C.free(unsafe.Pointer(s))

	fmt.Println("returning toerror: ", C.GoString(s))
	return err
}

func NewDecoder(r io.Reader) (*Decoder, error) {
	var e C.int
	mh := C.mpg123_new(nil, &e)
	if mh == nil || e != 0 {
		return nil, toError(e)
	}

	err := toError(C.mpg123_open_feed(mh))
	if err != nil {
		return nil, err
	}

	C.mpg123_format_none(mh)
	C.mpg123_format(mh, 44100, C.MPG123_STEREO, C.MPG123_ENC_FLOAT_32)

	buf := make([]byte, ReadBufferSize)
	return &Decoder{mh: mh, src: r, buf: buf}, nil
}

type Decoder struct {
	mh  *C.mpg123_handle
	src io.Reader
	buf []byte
}

func (d *Decoder) Read(p []byte) (n int, err error) {
	var stat C.int
loop:
	for {
		var rn int
		rn, err = d.src.Read(d.buf)
		if err != nil && err != io.EOF {
			return n, err
		}

		var cn C.size_t
		stat = C.mpg123_decode(d.mh,
			(*C.uchar)(unsafe.Pointer(&d.buf[0])), C.size_t(rn),
			(*C.uchar)(unsafe.Pointer(&p[n])), C.size_t(len(p)-n),
			&cn,
		)

		n += int(cn)

		switch stat {
		case C.MPG123_NEED_MORE:
			// mpg123 is asking for more data, so loop around if
			// we haven't reached EOF in the reader yet.
			if err == io.EOF {
				// We've exhausted the io.Reader and cleaned all
				// the internal buffers of mpg123, so we can signal
				// a proper EOF to our caller.
				return n, io.EOF
			}
			// Otherwise we just want more data
			break
		case C.MPG123_NEW_FORMAT:
			// mpg123 is notifying us of a new format coming up
			var rate C.long
			var chans, enc C.int
			err := toError(C.mpg123_getformat(d.mh, &rate, &chans, &enc))
			if err != nil {
				log.Println("error while getting stream format:", err)
			}
			fmt.Println(rate, chans, enc)
			fallthrough
		default:
			break loop
		}
	}

	// Check for read error from the src
	if err != nil && err != io.EOF {
		return n, err
	}

	if stat != C.MPG123_OK && stat != C.MPG123_NEW_FORMAT {
		return n, toError(stat)
	}

	return n, nil
}