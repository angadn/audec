package mp3

import (
	"io"

	"github.com/angadn/audec"
)

func AudecDecoder(r io.Reader) (audec.Decoder, error) {
	d, err := NewDecoder(r)
	return d, err
}

func init() {
	audec.RegisterDecoder(audec.MP3, AudecDecoder)
}
