package main

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_buildPath(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "no params",
			args: args{
				path: "/test/noparams",
			},
			want: "/test/noparams",
		},
		{
			name: "single param",
			args: args{
				path: "/test/{message.id}",
			},
			want: "/test/:message.id",
		},
		{
			name: "multiple params",
			args: args{
				path: "/test/{message.id}/{message.id}",
			},
			want: "/test/:message.id/:message.id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, buildPath(tt.args.path), "buildPath(%v)", tt.args.path)
		})
	}
}

func TestRegex(t *testing.T) {
	s := "@Path ( \"demo\" ) "
	sub := regexp.MustCompile(`(?i)@([a-z_]*)\s*(.*)`).FindStringSubmatch(s)
	assert.Equal(t, "Path", sub[1])
	assert.Equal(t, "demo", sub[2][2:len(sub[2])-2])
}

func Test_buildAnnotation(t *testing.T) {
	type args struct {
		comment string
	}
	tests := []struct {
		name     string
		args     args
		wantAnno annotation
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			name: "no annotation",
			args: args{
				comment: "",
			},
			wantAnno: annotation{},
			wantErr:  assert.NoError,
		},
		{
			name: "with comment",
			args: args{
				comment: `
					// hello world
            `,
			},
			wantAnno: annotation{
				Comment: "hello world",
			},
			wantErr: assert.NoError,
		},
		{
			name: "with path annotation",
			args: args{
				comment: `
					// @Path("demo")
			`,
			},
			wantAnno: annotation{
				Path: "demo",
			},
			wantErr: assert.NoError,
		},
		{
			name: "with auth annotation",
			args: args{
				comment: `
					// @Auth
			`,
			},
			wantAnno: annotation{
				Auth: true,
			},
			wantErr: assert.NoError,
		},
		{
			name: "with custom annotation",
			args: args{
				comment: `
					// @Customs("cache,etag")
			`,
			},
			wantAnno: annotation{
				Customs: []string{"cache", "etag"},
			},
			wantErr: assert.NoError,
		},
		{
			name: "error build annotation",
			args: args{
				comment: `
					// @RequestParam("name")
			`,
			},
			wantAnno: annotation{},
			wantErr:  assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAnno, err := buildAnnotation(tt.args.comment)
			if !tt.wantErr(t, err, fmt.Sprintf("buildAnnotation(%v)", tt.args.comment)) {
				return
			}
			assert.Equalf(t, tt.wantAnno, gotAnno, "buildAnnotation(%v)", tt.args.comment)
		})
	}
}
