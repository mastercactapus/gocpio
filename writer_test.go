package cpio

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"
	"time"
)

func testWriterType(t *testing.T, file string, enc EncodingType) {
	t.Run(enc.String(), func(t *testing.T) {

		hdr := &Header{
			Encoding: enc,
			DevMinor: 44,
			Inode:    1337,
			UID:      1000,
			GID:      1000,
			NLink:    1,
			Mode:     33204,
			Size:     6,
			Name:     "hello.txt",
			ModTime:  time.Unix(1337, 0),
		}

		if enc == EncodingTypeASCIISVR4CRC {
			hdr.Checksum = 562
		}

		data, err := ioutil.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}

		buf := new(bytes.Buffer)

		w := NewWriter(buf)
		err = w.WriteHeader(hdr)
		if err != nil {
			t.Fatal(err)
		}

		io.WriteString(w, "world\n")
		w.Close()

		tmp := make([]byte, len(data))
		copy(tmp, buf.Bytes())

		if !bytes.Equal(data, tmp) {
			if enc == EncodingTypeBinaryBE || enc == EncodingTypeBinaryLE {
				t.Errorf("Bad Output:\nExpected: %x\nActual:   %x", data, tmp)
			} else {
				t.Errorf("Bad Output:\nExpected: %s\nActual:   %s", data, tmp)
			}
		}

	})
}

func TestWriter(t *testing.T) {
	testWriterType(t, "test-data/ascii-susv2.cpio", EncodingTypeASCIISUSv2)
	testWriterType(t, "test-data/ascii-svr4.cpio", EncodingTypeASCIISVR4)
	testWriterType(t, "test-data/ascii-svr4-crc.cpio", EncodingTypeASCIISVR4CRC)
	testWriterType(t, "test-data/binary.cpio", EncodingTypeBinaryLE)
}
