package main

import (
	"bytes"
	"io"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type templateWithError struct{}

func (t *templateWithError) entry() string {
	return "{{.Undefined}}"
}

func (t *templateWithError) list() string {
	return ""
}

func Test_Lu_entriesByReq(t *testing.T) {
	entries := []*entry{{Request: "bcd"}, {Request: "abc"}}
	sort.Sort(entriesByReq(entries))
	assert.Equal(t, []*entry{{Request: "abc"}, {Request: "bcd"}}, entries)
}

func Test_Lu_writeFile(t *testing.T) {
	withSetup := func(setupFn func(lu *Lu), assertsFn func(result string, err error)) {
		lu := &Lu{}
		lu.history = []*entry{
			{Request: "dog", Responses: []*response{{Lang: "de", Translations: []string{"Hund", "Rüde"}}}},
			{Request: "cat", Responses: []*response{{Lang: "de", Translations: []string{"Katze"}}}},
			{Request: "pig", Responses: []*response{{Lang: "de", Translations: []string{"Schwein"}}}},
			{Request: "horse", Responses: []*response{{Lang: "de", Translations: []string{"Pferd", "Ross"}}}},
		}
		lu.fileTemplater = &textTemplater{}
		r, w, _ := os.Pipe()
		defer r.Close()
		defer w.Close()
		lu.dstFile = w

		if setupFn != nil {
			setupFn(lu)
		}

		err := lu.writeFile()
		w.Close()

		var b bytes.Buffer
		io.Copy(&b, r)
		assertsFn(b.String(), err)
	}

	withSetup(nil, func(result string, err error) {
		require.NoError(t, err)
		assert.Contains(t, result, "********")
		firstStr := strings.Split(result, "\n")[0]
		assert.Equal(t, "dog", firstStr)
		assert.Contains(t, result, "Hund")
	})

	withSetup(func(lu *Lu) {
		lu.fileTemplater = &htmlTemplater{}
	}, func(result string, err error) {
		require.NoError(t, err)
		assert.Contains(t, result, "<html>")
		assert.Contains(t, result, "dog")
		assert.Contains(t, result, "Hund")
	})

	withSetup(func(lu *Lu) {
		lu.opts.Sort = true
	}, func(result string, err error) {
		assert.Contains(t, result, "********")
		require.NoError(t, err)
		firstStr := strings.Split(result, "\n")[0]
		assert.Equal(t, "cat", firstStr)
	})

	withSetup(func(lu *Lu) {
		lu.history = nil
		lu.fileTemplater = &templateWithError{}
	}, func(result string, err error) {
		assert.Error(t, err)
	})
}

func Test_Lu_close(t *testing.T) {
	r, w, _ := os.Pipe()
	lu := &Lu{srcFile: r, dstFile: w}
	lu.close()
	assert.Nil(t, lu.srcFile)
	assert.Nil(t, lu.dstFile)
}

func Test_Lu_setupAPI(t *testing.T) {
	lu := &Lu{}
	oldD := os.Getenv("LU_YANDEX_DICTIONARY_API_KEY")
	oldT := os.Getenv("LU_YANDEX_TRANSLATE_API_KEY")
	defer os.Setenv("LU_YANDEX_TRANSLATE_API_KEY", oldT)
	defer os.Setenv("LU_YANDEX_DICTIONARY_API_KEY", oldD)

	os.Unsetenv("LU_YANDEX_DICTIONARY_API_KEY")
	err := lu.setupAPI()
	assert.EqualError(t, err, "the required environment variable LU_YANDEX_DICTIONARY_API_KEY is not set")

	os.Setenv("LU_YANDEX_DICTIONARY_API_KEY", "stub")

	os.Unsetenv("LU_YANDEX_TRANSLATE_API_KEY")
	err = lu.setupAPI()
	assert.EqualError(t, err, "the required environment variable LU_YANDEX_TRANSLATE_API_KEY is not set")

	os.Setenv("LU_YANDEX_TRANSLATE_API_KEY", "stub")

	err = lu.setupAPI()
	require.NoError(t, err)
	assert.NotNil(t, lu.dictionary)
	assert.NotNil(t, lu.translator)
}

func Test_Lu_setupInput(t *testing.T) {
	lu := &Lu{}
	r, _ := lu.setupInput([]string{})
	assert.Equal(t, os.Stdin, r)

	r, _ = lu.setupInput([]string{"hot", "dog"})
	assert.Equal(t, strings.NewReader("hot dog"), r)

	os.Create("tmp.txt")
	lu = &Lu{opts: options{SrcFileName: "tmp.txt"}}
	r, err := lu.setupInput([]string{})
	require.NoError(t, err)
	assert.NotNil(t, r)
	os.Remove("tmp.txt")

	lu = &Lu{opts: options{SrcFileName: "not_existed.txt"}}
	lu.setupInput([]string{})
	_, err = os.Open(lu.opts.SrcFileName)
	require.Error(t, err)
}

func Test_Lu_setupOutput(t *testing.T) {
	lu := &Lu{}

	err := lu.setupOutput()
	require.NoError(t, err)

	assert.Equal(t, &textTemplater{}, lu.stdoutTemplater)

	fname := "out.txt"
	os.Create(fname)
	lu = &Lu{opts: options{DstFileName: fname}}
	err = lu.setupOutput()
	require.NoError(t, err)
	assert.Equal(t, &textTemplater{}, lu.fileTemplater)
	os.Remove(fname)

	fname = "out.html"
	os.Create(fname)
	lu = &Lu{opts: options{DstFileName: fname}}
	err = lu.setupOutput()
	require.NoError(t, err)
	assert.Equal(t, &htmlTemplater{}, lu.fileTemplater)
	os.Remove(fname)
}
