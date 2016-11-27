package cpio

import (
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

var testModTime = time.Unix(1337, 0)

func intEq(t *testing.T, name string, expected, actual int) {
	if actual != expected {
		t.Errorf("expected %s to be %d but got %d", name, expected, actual)
	}
}

func testReaderType(t *testing.T, file string, enc EncodingType) *Header {
	var fileHeader *Header
	t.Run(enc.String(), func(t *testing.T) {

		t.Log("Testing:", file, "as", enc.String())
		fd, err := os.Open(file)
		if err != nil {
			t.Fatal("open test archive,", file, ":", err)
		}
		defer fd.Close()

		r := NewReader(fd)

		hdr, err := r.Next()
		if err != nil {
			t.Fatal("read first header:", err)
		}
		fileHeader = hdr

		intEq(t, "DevMajor", 0, hdr.DevMajor)
		intEq(t, "DevMinor", 44, hdr.DevMinor)
		intEq(t, "Inode", 1337, hdr.Inode)
		intEq(t, "UID", 1000, hdr.UID)
		intEq(t, "GID", 1000, hdr.GID)
		intEq(t, "NLink", 1, hdr.NLink)
		intEq(t, "RDevMajor", 0, hdr.RDevMajor)
		intEq(t, "RDevMinor", 0, hdr.RDevMinor)
		intEq(t, "Mode", 33204, int(hdr.Mode))
		intEq(t, "Size", 6, int(hdr.Size))

		if hdr.ModTime.Unix() != 1337 {
			t.Errorf("expected ModTime to be %d but got %d", 1337, hdr.ModTime.Unix())
		}

		if hdr.Encoding != enc {
			t.Errorf("expected Encoding to be %s but got %s", enc.String(), hdr.Encoding.String())
		}

		data, err := ioutil.ReadAll(r)
		if err != nil {
			t.Fatal("read file:", err)
		}

		if string(data) != "world\n" {
			t.Errorf("expected data to be '%s' but got '%s'", "world\n", string(data))
		}

		hdr, err = r.Next()
		if err != io.EOF {
			t.Error("expected io.EOF after last entry but got:", err, hdr)
		}
	})
	return fileHeader
}

func TestReader(t *testing.T) {
	testReaderType(t, "test-data/ascii-susv2.cpio", EncodingTypeASCIISUSv2)
	testReaderType(t, "test-data/ascii-svr4.cpio", EncodingTypeASCIISVR4)
	hdr := testReaderType(t, "test-data/ascii-svr4-crc.cpio", EncodingTypeASCIISVR4CRC)
	intEq(t, "Checksum", 562, hdr.Checksum)
	testReaderType(t, "test-data/binary.cpio", EncodingTypeBinaryLE)

}
