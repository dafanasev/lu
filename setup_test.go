package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseOpts(t *testing.T) {
	opts = options{}
	os.Args = []string{"lu"}

	oldF := os.Getenv("LU_DEFAULT_FROM_LANG")
	os.Setenv("LU_DEFAULT_FROM_LANG", "en")
	defer os.Setenv("LU_DEFAULT_FROM_LANG", oldF)

	oldT := os.Getenv("LU_DEFAULT_TO_LANGS")
	os.Setenv("LU_DEFAULT_TO_LANGS", "sp:fr")
	defer os.Setenv("LU_DEFAULT_TO_LANGS", oldT)

	parseOpts()

	assert.Equal(t, "en", opts.FromLang)
	assert.Equal(t, []string{"sp", "fr"}, opts.ToLangs)
	assert.False(t, opts.Sort)
	assert.False(t, opts.ShowLangs)
	assert.False(t, opts.Version)

	opts = options{}
	os.Args = []string{"lu", "-ffr", "-tru", "-tit", "-tde", "-iin.txt", "-oout.html", "-s", "-v", "-l", "hot dog"}
	args, err := parseOpts()
	assert.Equal(t, "fr", opts.FromLang)
	assert.Equal(t, []string{"ru", "it", "de"}, opts.ToLangs)
	assert.Equal(t, "in.txt", opts.SrcFileName)
	assert.Equal(t, "out.html", opts.DstFileName)
	assert.True(t, opts.Sort)
	assert.True(t, opts.ShowLangs)
	assert.True(t, opts.Version)
	assert.NoError(t, err)
	assert.Equal(t, []string{"hot dog"}, args)

	opts = options{}
	os.Args = append(os.Args, "-h")
	_, err = parseOpts()
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "can not parse arguments")

	opts = options{}
	os.Args = []string{"lu", "-ilist.txt", "-olist.txt"}
	_, err = parseOpts()
	require.Error(t, err)
	assert.EqualError(t, err, "source and destination must be different files")

	opts = options{}
	os.Args = []string{"lu", "-e"}
	_, err = parseOpts()
	require.Equal(t, "", opts.SrcFileName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "can not parse arguments")
}

func Test_setupInput(t *testing.T) {
	r, _ := setupInput([]string{})
	assert.Equal(t, os.Stdin, r)

	r, _ = setupInput([]string{"hot", "dog"})
	assert.Equal(t, strings.NewReader("hot dog"), r)

	opts = options{}
	os.Create("tmp.txt")
	opts.SrcFileName = "tmp.txt"
	r, err := setupInput([]string{})
	require.NoError(t, err)
	assert.NotNil(t, r)
	os.Remove("tmp.txt")

	opts = options{}
	opts.SrcFileName = "not_existed.txt"
	setupInput([]string{})
	_, err = os.Open(opts.SrcFileName)
	require.Error(t, err)
}

func Test_setupFileOutput(t *testing.T) {
	opts = options{}

	err := setupFileOutput()
	require.NoError(t, err)

	fname := "out.txt"
	os.Create(fname)
	opts.DstFileName = fname
	err = setupFileOutput()
	require.NoError(t, err)
	assert.Equal(t, &textTemplater{}, fileTemplater)
	os.Remove(fname)

	fname = "out.html"
	os.Create(fname)
	opts.DstFileName = fname
	err = setupFileOutput()
	require.NoError(t, err)
	assert.Equal(t, &htmlTemplater{}, fileTemplater)
	os.Remove(fname)
}

func Test_setup(t *testing.T) {
	opts = options{}
	os.Args = []string{"lu"}
	oldD := os.Getenv("LU_YANDEX_DICTIONARY_API_KEY")
	oldT := os.Getenv("LU_YANDEX_TRANSLATE_API_KEY")
	defer os.Setenv("LU_YANDEX_TRANSLATE_API_KEY", oldT)
	defer os.Setenv("LU_YANDEX_DICTIONARY_API_KEY", oldD)

	os.Unsetenv("LU_YANDEX_DICTIONARY_API_KEY")
	err := setup([]string{})
	assert.EqualError(t, err, "the required environment variable LU_YANDEX_DICTIONARY_API_KEY is not set")

	os.Setenv("LU_YANDEX_DICTIONARY_API_KEY", "stub")

	os.Unsetenv("LU_YANDEX_TRANSLATE_API_KEY")
	err = setup([]string{})
	assert.EqualError(t, err, "the required environment variable LU_YANDEX_TRANSLATE_API_KEY is not set")

	os.Setenv("LU_YANDEX_TRANSLATE_API_KEY", "stub")

	err = setup([]string{})
	require.NoError(t, err)
	assert.NotNil(t, dict)
	assert.NotNil(t, tr)
}
