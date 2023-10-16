package cli_test

import (
	"errors"
	"flag"
	"io"
	"reflect"
	"regexp"
	"testing"

	"github.com/michenriksen/pkgdmp"
	"github.com/michenriksen/pkgdmp/internal/cli"
)

func TestParseFlags(t *testing.T) {
	tt := []struct {
		name         string
		args         []string
		wantCfg      *cli.Config
		wantExitCode int
		wantErr      error
	}{
		{
			name:         "no args",
			wantExitCode: 1,
			wantErr:      cli.ErrNoDirs,
		},
		{
			name:         "help flag",
			args:         []string{"-help"},
			wantExitCode: 0,
			wantErr:      flag.ErrHelp,
		},
		{
			name:         "version flag",
			args:         []string{"-version"},
			wantExitCode: 0,
			wantErr:      cli.ErrVersion,
		},
		{
			name:         "flags but no directories",
			args:         []string{"-unexported", "-full-doc"},
			wantExitCode: 1,
			wantErr:      cli.ErrNoDirs,
		},
		{
			name: "flags and directories",
			args: []string{"-unexported", "-no-doc", "-no-interfaces", "directory1", "directory2"},
			wantCfg: &cli.Config{
				Unexported:   true,
				NoDoc:        true,
				NoInterfaces: true,
				Dirs:         []string{"directory1", "directory2"},
				Theme:        "swapoff",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			cfg, exitCode, err := cli.ParseFlags(tc.args, io.Discard)

			if !reflect.DeepEqual(cfg, tc.wantCfg) {
				t.Errorf("expected config:\n\n%#v\n\nbut got:\n\n%#v", tc.wantCfg, cfg)
			}

			if exitCode != tc.wantExitCode {
				t.Errorf("expected exit code %d, but got %d", tc.wantExitCode, exitCode)
			}

			if !errors.Is(err, tc.wantErr) {
				if tc.wantErr == nil {
					t.Errorf("expected no error, but got: %v", err)
				}

				t.Errorf("expected error %v, but got: %v", tc.wantErr, err)
			}
		})
	}
}

func TestParserOptsFromCfg(t *testing.T) {
	tt := []struct {
		name          string
		cfg           *cli.Config
		wantOpts      pkgdmp.ParserOptions
		wantErrRegexp *regexp.Regexp
	}{
		{
			name:     "default config",
			cfg:      &cli.Config{},
			wantOpts: pkgdmp.ParserOptions{},
		},
		{
			name:     "full docs and exclude interfaces",
			cfg:      &cli.Config{FullDoc: true, NoInterfaces: true},
			wantOpts: pkgdmp.ParserOptions{FullDocs: true, ExcludeInterfaces: true},
		},
		{
			name: "match and exclude patterns",
			cfg:  &cli.Config{Match: `^FooBa(r|z)`, Exclude: `(Hello|Hi)World`},
			wantOpts: pkgdmp.ParserOptions{
				OnlyRegexp:    regexp.MustCompile(`^FooBa(r|z)`),
				ExcludeRegexp: regexp.MustCompile(`(Hello|Hi)World`),
			},
		},
		{
			name:          "invalid match regexp",
			cfg:           &cli.Config{Match: `a\x{2`},
			wantErrRegexp: regexp.MustCompile(`parsing match regular expression:.*invalid escape sequence`),
		},
		{
			name:          "invalid exclude regexp",
			cfg:           &cli.Config{Exclude: `a\x{2`},
			wantErrRegexp: regexp.MustCompile(`parsing exclude regular expression:.*invalid escape sequence`),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := cli.ParserOptsFromCfg(tc.cfg)

			if !reflect.DeepEqual(opts, tc.wantOpts) {
				t.Errorf("expected options:\n\n%#v\n\nbut got:\n\n%#v\n\n", tc.wantOpts, opts)
			}

			if tc.wantErrRegexp != nil {
				if err == nil {
					t.Errorf("expected error matching regular expression `%s`, but got no error", tc.wantErrRegexp)
				}

				if !tc.wantErrRegexp.MatchString(err.Error()) {
					t.Errorf("expected error %q to match regular expression `%s`", err, tc.wantErrRegexp)
				}
			}
		})
	}
}
