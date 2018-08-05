package main

import (
	"bytes"
	"io"
	"os"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

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

func Test_entriesByReq(t *testing.T) {
	entries := []*entry{{Request: "bcd"}, {Request: "abc"}}
	sort.Sort(entriesByReq(entries))
	assert.Equal(t, []*entry{{Request: "abc"}, {Request: "bcd"}}, entries)
}

func Test_printResults(t *testing.T) {
	e := &entry{Request: "dog", Responses: []*response{{Lang: "de", Translations: []string{"Hund", "Rüde"}}}}

	printResultsWrapper := func(e *entry, data struct{ src, dst *os.File }) string {
		srcFile = data.src
		dstFile = data.dst
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		printResults(e, 1)

		resultChan := make(chan string)
		go func() {
			var b bytes.Buffer
			io.Copy(&b, r)
			r.Close()
			resultChan <- b.String()
		}()
		os.Stdout = old
		w.Close()
		return <-resultChan
	}

	cases := []struct{ src, dst *os.File }{
		{nil, nil},
		{os.Stdin, nil},
		{nil, os.Stdout},
	}
	for _, cs := range cases {
		result := printResultsWrapper(e, cs)
		assert.NotContains(t, result, "1. Got results")
		assert.Contains(t, result, "Rüde")
	}

	result := printResultsWrapper(e, struct{ src, dst *os.File }{src: os.Stdin, dst: os.Stdout})
	assert.Contains(t, result, "1. Got results")
	assert.NotContains(t, result, "Rüde")
}

func Test_handleExitSignal(t *testing.T) {
	opts = options{}
	os.Args = []string{"lu"}
	dstFile = nil

	go func() {
		time.Sleep(100 * time.Millisecond)
		err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		require.Nil(t, err)
	}()
	ts := time.Now()
	main()
	assert.True(t, time.Since(ts).Seconds() < 1)
}

func Test_writeFile(t *testing.T) {
	withSetup := func(setupFn func(), assertsFn func(result string, err error)) {
		opts = options{}
		history = []*entry{
			{Request: "dog", Responses: []*response{{Lang: "de", Translations: []string{"Hund", "Rüde"}}}},
			{Request: "cat", Responses: []*response{{Lang: "de", Translations: []string{"Katze"}}}},
			{Request: "pig", Responses: []*response{{Lang: "de", Translations: []string{"Schwein"}}}},
			{Request: "horse", Responses: []*response{{Lang: "de", Translations: []string{"Pferd", "Ross"}}}},
		}
		fileTemplater = &textTemplater{}
		r, w, _ := os.Pipe()
		defer r.Close()
		defer w.Close()
		dstFile = w

		if setupFn != nil {
			setupFn()
		}

		err := writeFile()
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

	withSetup(func() {
		fileTemplater = &htmlTemplater{}
	}, func(result string, err error) {
		require.NoError(t, err)
		assert.Contains(t, result, "<html>")
		assert.Contains(t, result, "dog")
		assert.Contains(t, result, "Hund")
	})

	withSetup(func() {
		opts.Sort = true
	}, func(result string, err error) {
		assert.Contains(t, result, "********")
		require.NoError(t, err)
		firstStr := strings.Split(result, "\n")[0]
		assert.Equal(t, "cat", firstStr)
	})

	withSetup(func() {
		history = nil
		fileTemplater = &templateWithError{}
	}, func(result string, err error) {
		assert.Error(t, err)
	})
}
