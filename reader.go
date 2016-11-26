package cpio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"time"
)

var (
	// ErrHeader is returned if the header was unable to be decoded
	ErrHeader = errors.New("github.com/mastercactapus/gocpio: invalid cpio header")
)

// A Reader provides sequential access to the contents of a cpio archive.
type Reader struct {
	r     io.Reader
	err   error
	lr    io.Reader
	buf   []byte
	align int
}

// NewReader creates a new Reader reading from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{r: r, buf: make([]byte, 0, 32768)}
}

// Read reads from the current entry in the cpio archive.
//
// It returns 0, io.EOF when it reaches the end of that entry,
// until Next is called to advance to the next entry.
func (cr *Reader) Read(b []byte) (int, error) {
	if cr.err != nil {
		return 0, cr.err
	}
	if cr.lr == nil {
		return 0, io.EOF
	}
	n, err := cr.lr.Read(b)
	if err != nil {
		if err != io.EOF {
			cr.err = err
		} else {
			cr.lr = nil
		}
	}
	return n, err
}

// Next advances to the next entry in the cpio archive.
//
// io.EOF is returned at the end of the input.
func (cr *Reader) Next() (*Header, error) {
	if cr.err != nil {
		return nil, cr.err
	}

	if cr.lr != nil {
		// skip through current file data
		_, cr.err = io.Copy(ioutil.Discard, cr)
		if cr.err != nil {
			return nil, cr.err
		}
	}

	cr.buf = cr.buf[:2+cr.align]
	_, cr.err = io.ReadFull(cr.r, cr.buf)
	if cr.err != nil {
		return nil, cr.err
	}
	cr.buf = cr.buf[cr.align:]

	switch {
	case bytes.Equal(cr.buf, []byte{0x71, 0xc7}): // binary, big-endian
		return cr.nextBinary(binary.BigEndian, EncodingTypeBinaryBE)
	case bytes.Equal(cr.buf, []byte{0xc7, 0x71}): // binary, little-endian
		return cr.nextBinary(binary.LittleEndian, EncodingTypeBinaryLE)
	case bytes.Equal(cr.buf, []byte{0x30, 0x37}): //ascii
		return cr.nextASCII()
	default:
		cr.err = ErrHeader
		return nil, cr.err
	}
}

func (cr *Reader) nextASCII() (*Header, error) {
	cr.buf = cr.buf[:4]
	_, cr.err = io.ReadFull(cr.r, cr.buf)
	if cr.err != nil {
		return nil, cr.err
	}
	switch {
	case bytes.Equal(cr.buf, []byte{0x30, 0x37, 0x30, 0x37}): // SUSv2
		return cr.nextASCIISUSv2()
	case bytes.Equal(cr.buf, []byte{0x30, 0x37, 0x30, 0x31}): // SVR4
		return cr.nextASCIISVR4(EncodingTypeASCIISVR4)
	case bytes.Equal(cr.buf, []byte{0x30, 0x37, 0x30, 0x32}): // SVR4CRC
		return cr.nextASCIISVR4(EncodingTypeASCIISVR4CRC)
	default:
		cr.err = ErrHeader
		return nil, cr.err
	}
}

func (cr *Reader) parseInt(dst *int, b []byte, base int) {
	if cr.err != nil {
		return
	}
	var i int64
	i, cr.err = strconv.ParseInt(string(b), base, 64)
	if cr.err != nil {
		return
	}
	*dst = int(i)
}
func (cr *Reader) parseInt64(dst *int64, b []byte, base int) {
	if cr.err != nil {
		return
	}
	*dst, cr.err = strconv.ParseInt(string(b), base, 64)
}

func (cr *Reader) nextASCIISUSv2() (*Header, error) {
	var modTime int64
	var nameSize int
	hdr := &Header{Encoding: EncodingTypeASCIISUSv2}
	_, cr.err = fmt.Fscanf(cr.r, "%06o%06o%06o%06o%06o%06o%06o%011o%06o%011o",
		&hdr.DevMinor,
		&hdr.Inode,
		&hdr.Mode,
		&hdr.UID,
		&hdr.GID,
		&hdr.NLink,
		&hdr.RDevMinor,
		&modTime,
		&nameSize,
		&hdr.Size,
	)
	hdr.ModTime = time.Unix(modTime, 0)

	return cr.nextName(hdr, nameSize)
}

func (cr *Reader) nextName(hdr *Header, p int) (*Header, error) {
	if cr.err != nil {
		return nil, cr.err
	}
	var rem int
	switch hdr.Encoding {
	case EncodingTypeBinaryLE, EncodingTypeBinaryBE:
		p += p % 2
		cr.align = int(hdr.Size % 2)
	case EncodingTypeASCIISVR4, EncodingTypeASCIISVR4CRC:
		rem = (p + 2) % 4
		if rem > 0 {
			p += 4 - rem
		}
		rem = int((hdr.Size + int64(rem)) % 4)
		if rem > 0 {
			cr.align = 4 - rem
		} else {
			cr.align = 0
		}
	default:
		cr.align = 0
	}

	if cap(cr.buf) < p {
		cr.buf = make([]byte, p)
	} else {
		cr.buf = cr.buf[:p]
	}

	_, cr.err = io.ReadFull(cr.r, cr.buf)
	if cr.err != nil {
		return nil, cr.err
	}
	p = bytes.IndexByte(cr.buf, 0)
	if p == -1 {
		hdr.Name = string(cr.buf)
	} else {
		hdr.Name = string(cr.buf[:p])
	}
	if hdr.Name == "TRAILER!!!" && hdr.Size == 0 {
		return nil, io.EOF
	}

	cr.lr = io.LimitReader(cr.r, hdr.Size)
	return hdr, nil
}

func (cr *Reader) nextASCIISVR4(encoding EncodingType) (*Header, error) {
	var modTime int64
	var nameSize int
	hdr := &Header{Encoding: encoding}
	_, cr.err = fmt.Fscanf(cr.r, "%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x",
		&hdr.Inode,
		&hdr.Mode,
		&hdr.UID,
		&hdr.GID,
		&hdr.NLink,
		&modTime,
		&hdr.Size,
		&hdr.DevMajor,
		&hdr.DevMinor,
		&hdr.RDevMajor,
		&hdr.RDevMinor,
		&nameSize,
		&hdr.Checksum,
	)
	hdr.ModTime = time.Unix(modTime, 0)

	return cr.nextName(hdr, nameSize)
}

func (cr *Reader) nextBinary(order binary.ByteOrder, enc EncodingType) (*Header, error) {
	var h binaryHeader
	cr.err = binary.Read(cr.r, order, &h)
	if cr.err != nil {
		return nil, cr.err
	}

	hdr := &Header{
		Encoding:  enc,
		DevMinor:  int(h.Dev),
		Inode:     int(h.Inode),
		Mode:      int64(h.Mode),
		UID:       int(h.UID),
		GID:       int(h.GID),
		NLink:     int(h.NLink),
		RDevMinor: int(h.RDev),
		ModTime:   time.Unix(65536*int64(h.ModTime[0])+int64(h.ModTime[1]), 0),
		Size:      65536*int64(h.Filesize[0]) + int64(h.Filesize[1]),
	}

	return cr.nextName(hdr, int(h.Namesize))
}
