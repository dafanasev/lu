package main

import (
	"bytes"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain unsets LU_* environment variables before running test suite
// to get clean test environment and restores them after running
func TestMain(m *testing.M) {
	keys := []string{"LU_YANDEX_DICTIONARY_API_KEY", "LU_YANDEX_TRANSLATE_API_KEY", "LU_DEFAULT_FROM_LANG", "LU_DEFAULT_TO_LANGS"}
	envVars := make(map[string]string, len(keys))
	for _, k := range keys {
		envVars[k] = os.Getenv(k)
		os.Unsetenv(k)
	}

	code := m.Run()

	for _, k := range keys {
		os.Setenv(k, envVars[k])
	}

	os.Exit(code)
}

func Test_parseCommandLine(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	for _, flag := range []string{"", "-fen", "-tde"} {
		os.Args = append([]string{"lu"}, flag)
		_, _, err := parseCommandLine()
		assert.EqualError(t, err, "translation direction (-f and -t flags must be specified")
	}
	for _, flag := range []string{"-v", "-l"} {
		os.Args = append([]string{"lu"}, flag)
		_, _, err := parseCommandLine()
		assert.NoError(t, err)
	}

	os.Args = []string{"lu"}

	oldF := os.Getenv("LU_DEFAULT_FROM_LANG")
	os.Setenv("LU_DEFAULT_FROM_LANG", "en")
	defer os.Setenv("LU_DEFAULT_FROM_LANG", oldF)

	oldT := os.Getenv("LU_DEFAULT_TO_LANGS")
	os.Setenv("LU_DEFAULT_TO_LANGS", "sp:fr")
	defer os.Setenv("LU_DEFAULT_TO_LANGS", oldT)

	_, opts, _ := parseCommandLine()

	assert.Equal(t, "en", opts.FromLang)
	assert.Equal(t, []string{"sp", "fr"}, opts.ToLangs)
	assert.False(t, opts.Sort)
	assert.False(t, opts.ShowLangs)
	assert.False(t, opts.Version)

	os.Args = []string{"lu", "-ffr", "-tru", "-tit", "-tde", "-iin.txt", "-oout.html", "-s", "-v", "-l", "hot dog"}
	args, opts, err := parseCommandLine()
	assert.Equal(t, "fr", opts.FromLang)
	assert.Equal(t, []string{"ru", "it", "de"}, opts.ToLangs)
	assert.Equal(t, "in.txt", opts.SrcFileName)
	assert.Equal(t, "out.html", opts.DstFileName)
	assert.True(t, opts.Sort)
	assert.True(t, opts.ShowLangs)
	assert.True(t, opts.Version)
	assert.NoError(t, err)
	assert.Equal(t, []string{"hot dog"}, args)

	os.Args = append(os.Args, "-h")
	_, opts, err = parseCommandLine()
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "can not parse arguments")

	os.Args = []string{"lu", "-ilist.txt", "-olist.txt"}
	_, opts, err = parseCommandLine()
	require.Error(t, err)
	assert.EqualError(t, err, "source and destination must be different files")

	os.Args = []string{"lu", "-e"}
	_, opts, err = parseCommandLine()
	require.Equal(t, "", opts.SrcFileName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "can not parse arguments")
}

func Test_printResults(t *testing.T) {
	e := &entry{Request: "dog", Responses: []*response{{Lang: "de", Translations: []string{"Hund", "Rüde"}}}}

	// write to buffer instead of stdout so we can test output
	printResultsWrapper := func(e *entry, data struct{ src, dst *os.File }) string {
		lu := &Lu{}
		lu.srcFile = data.src
		lu.dstFile = data.dst
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		printResults(lu, e, 1)

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
		result := printResultsWrapper(e, struct{ src, dst *os.File }{src: cs.src, dst: cs.dst})
		assert.NotContains(t, result, "1. Got results")
		assert.Contains(t, result, "Rüde")
	}

	result := printResultsWrapper(e, struct{ src, dst *os.File }{src: os.Stdin, dst: os.Stdout})
	assert.Contains(t, result, "1. Got results")
	assert.NotContains(t, result, "Rüde")

	// testing error in the template, app should exit with code = 1
	// in order to test it, run app in the separate process
	old := stdoutTemplater
	stdoutTemplater = func() *template.Template {
		return template.Must(template.New("").Funcs(templatesFnMap).Parse((&textTemplater{}).entry() + "{{ .Undefined }}\n"))
	}()

	if os.Getenv("LU_CRASH") == "1" {
		printResults(&Lu{}, e, 1)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=Test_printResults")
	cmd.Env = append(os.Environ(), "LU_CRASH=1")
	err := cmd.Run()
	exitError, ok := err.(*exec.ExitError)
	assert.True(t, ok && !exitError.Success())

	stdoutTemplater = old
}

func Test_handleExitSignal(t *testing.T) {
	oldArgs := os.Args
	os.Setenv("LU_TEST", "1")
	defer func() {
		os.Args = oldArgs
		os.Unsetenv("LU_TEST")
	}()
	os.Args = []string{"lu", "-fen", "-tde"}

	go func() {
		time.Sleep(100 * time.Millisecond)
		err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		require.Nil(t, err)
	}()
	ts := time.Now()
	main()
	assert.True(t, time.Since(ts).Seconds() < 1)
}

func Test_Main(t *testing.T) {
	mainWrapper := func() string {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		main()

		resultCh := make(chan string)
		go func() {
			var b bytes.Buffer
			io.Copy(&b, r)
			r.Close()
			resultCh <- b.String()
		}()
		os.Stdout = old
		w.Close()
		return <-resultCh
	}

	oldArgs := os.Args
	os.Setenv("LU_TEST", "1")
	defer func() {
		os.Args = oldArgs
		os.Unsetenv("LU_TEST")
	}()

	os.Args = []string{"lu", "-v"}
	result := mainWrapper()
	assert.Contains(t, result, "lu")

	os.Args = []string{"lu", "-l"}
	result = mainWrapper()
	assert.Contains(t, result, "Supported languages:")

	os.Args = []string{"lu", "-fen", "-tde", "black dog"}
	result = mainWrapper()
	assert.Contains(t, result, "schwarzer Hund")

	os.Args = []string{"lu", "-fen", "-tde", "-oout.txt", "black dog"}
	mainWrapper()
	fcontents, _ := ioutil.ReadFile("out.txt")
	os.Remove("out.txt")
	assert.Contains(t, string(fcontents), "schwarzer Hund")
}
