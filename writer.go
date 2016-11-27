package cpio

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

var (
	ErrWriteAfterClose = errors.New("cpio: write after close")
	ErrWriteTooLong    = errors.New("cpio: write too long")
)

var zeroBlock = make([]byte, 4)

// A Writer provides sequential writing of a cpio archive.
// Call WriteHeader to begin a new file, and then call Write to supply
// that file's data, writing at most hdr.Size bytes in total.
type Writer struct {
	w      io.Writer
	err    error
	closed bool
	nb     int64
	pad    int64
	first  bool
	enc    EncodingType
	hdrBuf []byte
}

// NewWriter creates a new Writer writing to w
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// Close closes the cpio archive, flushing any unwritten data to the underlying writer.
func (cw *Writer) Close() error {
	if cw.err != nil || cw.closed {
		return cw.err
	}

	cw.WriteHeader(&Header{
		Encoding: cw.enc,
		Name:     "TRAILER!!!",
		NLink:    1,
		ModTime:  time.Unix(0, 0),
	})

	cw.Flush()
	cw.closed = true

	return cw.err
}

// Flush finishes writing the current file (optional).
func (cw *Writer) Flush() error {
	if cw.nb > 0 {
		cw.err = fmt.Errorf("cpio: missed writing %d bytes", cw.nb)
		return cw.err
	}
	if cw.pad == 0 {
		return cw.err
	}
	_, cw.err = cw.w.Write(zeroBlock[:cw.pad])
	cw.pad = 0
	return cw.err
}

// Write writes to the current entry in the tar archive.
// Write returns the error ErrWriteTooLong if more than
// hdr.Size bytes are written after WriteHeader.
func (cw *Writer) Write(b []byte) (int, error) {
	if cw.closed {
		return 0, ErrWriteAfterClose
	}
	overwrite := false
	if int64(len(b)) > cw.nb {
		b = b[:cw.nb]
		overwrite = true
	}
	n, err := cw.w.Write(b)
	cw.nb -= int64(n)
	if err == nil && overwrite {
		return n, ErrWriteTooLong
	}
	cw.err = err
	return n, err
}

// WriteHeader writes hdr and prepares to accept the file's contents.
// WriteHeader calls Flush if it is not the first header. Calling
// after a Close will return ErrWriteAfterClose.
func (cw *Writer) WriteHeader(hdr *Header) error {
	if cw.closed {
		return ErrWriteAfterClose
	}
	if cw.err == nil {
		cw.Flush()
	}

	// flush could have set an error, so don't use `else`
	if cw.err != nil {
		return cw.err
	}
	if !cw.first {
		cw.first = true
		cw.enc = hdr.Encoding
	}

	// TODO: what happens if we get different header formats?

	switch hdr.Encoding {
	case EncodingTypeBinaryBE:
		return cw.writeBinary(hdr, binary.BigEndian)
	case EncodingTypeBinaryLE:
		return cw.writeBinary(hdr, binary.LittleEndian)
	case EncodingTypeASCIISUSv2:
		return cw.nextASCIISUSv2(hdr)
	case EncodingTypeASCIISVR4, EncodingTypeASCIISVR4CRC:
		return cw.nextASCIISVR4(hdr)
	default:
		return fmt.Errorf("cpio: unknown header encoding type")
	}
}

func (cw *Writer) nextASCIISVR4(hdr *Header) error {
	nameLen := len(hdr.Name) + 1
	var namePad string
	rem := (nameLen + 2) % 4
	if rem > 0 {
		namePad = strings.Repeat("\x00", 4-rem)
	}
	_, cw.err = fmt.Fprintf(cw.w, "07070%d%08X%08X%08X%08X%08X%08X%08X%08X%08X%08X%08X%08X%08X%s\x00%s",
		hdr.Encoding,
		hdr.Inode,
		hdr.Mode,
		hdr.UID,
		hdr.GID,
		hdr.NLink,
		hdr.ModTime.Unix(),
		hdr.Size,
		hdr.DevMajor,
		hdr.DevMinor,
		hdr.RDevMajor,
		hdr.RDevMinor,
		nameLen,
		hdr.Checksum,
		hdr.Name,
		namePad,
	)

	cw.pad = hdr.Size % 4
	if cw.pad > 0 {
		cw.pad = 4 - cw.pad
	}
	cw.nb = hdr.Size

	return cw.err
}

func (cw *Writer) nextASCIISUSv2(hdr *Header) error {
	_, cw.err = fmt.Fprintf(cw.w, "070707%06o%06o%06o%06o%06o%06o%06o%011o%06o%011o%s\x00",
		hdr.DevMinor,
		hdr.Inode,
		hdr.Mode,
		hdr.UID,
		hdr.GID,
		hdr.NLink,
		hdr.RDevMinor,
		hdr.ModTime.Unix(),
		len(hdr.Name)+1,
		hdr.Size,
		hdr.Name,
	)
	cw.pad = 0
	cw.nb = hdr.Size
	return cw.err
}

func (cw *Writer) writeBinary(hdr *Header, bo binary.ByteOrder) error {
	cw.err = binary.Write(cw.w, bo, uint16(070707))
	if cw.err != nil {
		return cw.err
	}

	var h binaryHeader
	h.Dev = uint16(hdr.DevMinor)
	h.Filesize[0] = uint16(hdr.Size / 65536)
	h.Filesize[1] = uint16(hdr.Size % 65536)
	h.GID = uint16(hdr.GID)
	h.Inode = uint16(hdr.Inode)
	h.Mode = uint16(hdr.Mode)
	mt := hdr.ModTime.Unix()
	h.ModTime[0] = uint16(mt / 65536)
	h.ModTime[1] = uint16(mt % 65536)
	nlen := len(hdr.Name) + 1
	h.Namesize = uint16(nlen)
	h.NLink = uint16(hdr.NLink)
	h.RDev = uint16(hdr.RDevMinor)
	h.UID = uint16(hdr.UID)

	cw.err = binary.Write(cw.w, bo, &h)
	if cw.err != nil {
		return cw.err
	}

	nameBuf := make([]byte, nlen+nlen%2)
	copy(nameBuf, hdr.Name)
	_, cw.err = cw.w.Write(nameBuf)
	if cw.err != nil {
		return cw.err
	}

	cw.nb = hdr.Size
	cw.pad = hdr.Size % 2
	return nil
}
