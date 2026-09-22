package models

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/check.v1"
)

func (s *ModelsSuite) TestAttachment(c *check.C) {
	ptx := PhishingTemplateContext{
		BaseRecipient: BaseRecipient{
			FirstName: "Foo",
			LastName:  "Bar",
			Email:     "foo@bar.com",
			Position:  "Space Janitor",
		},
		BaseURL:     "http://testurl.com",
		URL:         "http://testurl.com/?rid=1234567",
		TrackingURL: "http://testurl.local/track?rid=1234567",
		Tracker:     "<img alt='' style='display: none' src='http://testurl.local/track?rid=1234567'/>",
		From:        "From Address",
		RId:         "1234567",
	}

	files, err := ioutil.ReadDir("testdata")
	if err != nil {
		log.Fatalf("Failed to open attachment folder 'testdata': %v\n", err)
	}
	for _, ff := range files {
		if !ff.IsDir() && !strings.Contains(ff.Name(), "templated") {
			fname := ff.Name()
			fmt.Printf("Checking attachment file -> %s\n", fname)
			data := readFile("testdata/" + fname)
			if filepath.Ext(fname) == ".b64" {
				fname = fname[:len(fname)-4]
			}
			a := Attachment{
				Content: data,
				Name:    fname,
			}
			t, err := a.ApplyTemplate(ptx)
			c.Assert(err, check.Equals, nil)
			c.Assert(a.vanillaFile, check.Equals, strings.Contains(fname, "without-vars"))
			c.Assert(a.vanillaFile, check.Not(check.Equals), strings.Contains(fname, "with-vars"))

			// Verify template was applied as expected
			tt, err := ioutil.ReadAll(t)
			if err != nil {
				log.Fatalf("Failed to parse templated file '%s': %v\n", fname, err)
			}
			expectedOutput := readFile("testdata/" + strings.TrimSuffix(ff.Name(), filepath.Ext(ff.Name())) + ".templated" + filepath.Ext(ff.Name())) // e.g text-file-with-vars.templated.txt
			expectedBytes, err := base64.StdEncoding.DecodeString(expectedOutput)
			c.Assert(err, check.IsNil)
			assertAttachmentOutputEqual(c, fname, tt, expectedBytes)
		}
	}
}

// assertAttachmentOutputEqual compares templated attachment bytes. Office
// documents are ZIP archives whose compressed bytes are not stable across Go
// versions, so those are compared by uncompressed member contents.
func assertAttachmentOutputEqual(c *check.C, name string, got, expected []byte) {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".docx", ".docm", ".pptx", ".xlsx", ".xlsm":
		assertZipContentsEqual(c, got, expected)
	default:
		c.Assert(got, check.DeepEquals, expected)
	}
}

func assertZipContentsEqual(c *check.C, got, expected []byte) {
	gotFiles, err := readZipMembers(got)
	c.Assert(err, check.IsNil)
	expFiles, err := readZipMembers(expected)
	c.Assert(err, check.IsNil)
	c.Assert(len(gotFiles), check.Equals, len(expFiles))
	for name, exp := range expFiles {
		actual, ok := gotFiles[name]
		c.Assert(ok, check.Equals, true)
		c.Assert(actual, check.DeepEquals, exp)
	}
}

func readZipMembers(data []byte) (map[string][]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	out := make(map[string][]byte, len(zr.File))
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		contents, err := ioutil.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		out[f.Name] = contents
	}
	return out, nil
}

func readFile(fname string) string {
	f, err := os.Open(fname)
	if err != nil {
		log.Fatalf("Failed to open file '%s': %v\n", fname, err)
	}
	reader := bufio.NewReader(f)
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Fatalf("Failed to read file '%s': %v\n", fname, err)
	}
	data := ""
	if filepath.Ext(fname) == ".b64" {
		data = string(content)
	} else {
		data = base64.StdEncoding.EncodeToString(content)
	}
	return data
}
