package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Context struct {
	context.Context
}

var CLI struct {
	Profile Profile `cmd:"" aliases:"p,prof" help:"run a profile and upload"`
	Upload  Upload  `cmd:"" aliases:"u,up" help:"upload a file"`
}

var httpClient = &http.Client{}

type BaseArgs struct {
	Remote *url.URL `name:"remote" help:"which remote pprofweb server to use" env:"PPROFWEB_REMOTE_URL" default:"https://pprof.tuxpa.in"`
}

type Profile struct {
	Mode string `name:"mode" short:"m" enum:"heap,goroutine,threadcreate,block,mutex,profile" help:"type of profile to run, heap,goroutine,threadcreate,block,mutex,profile" default:"heap"`
	Time int    `name:"time" short:"t" help:"duration of profile in seconds"`

	Url *url.URL `arg:"" help:"base url of pprof server, ala http://localhost:6060/debug/pprof"`

	BaseArgs
}

func (c *Profile) Run(ctx Context) error {
	req := &http.Request{}
	req = req.WithContext(ctx)

	req.Method = "GET"
	req.URL = c.Url
	req.URL = req.URL.JoinPath(c.Mode)

	q := req.URL.Query()
	if c.Time > 0 {
		q.Add("seconds", strconv.Itoa(c.Time))
	}
	req.URL.RawQuery = q.Encode()

	fmt.Printf("getting %s profile from %s...\n", c.Mode, req.URL)
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	res, err := doUpload(ctx, c.BaseArgs, resp.Body)
	if err != nil {
		return err
	}
	fmt.Printf("see %s profile at %s/%s/ \n", c.Mode, c.Remote, res)
	return nil
}

type Upload struct {
	File string `arg:"" help:"file to upload" type:"path"`
	BaseArgs
}

func (c *Upload) Run(ctx Context) error {
	fl, err := os.Open(c.File)
	if err != nil {
		return err
	}
	defer fl.Close()
	res, err := doUpload(ctx, c.BaseArgs, fl)
	if err != nil {
		return err
	}
	fmt.Printf("see profile at %s/%s/ \n", c.Remote, res)
	return nil
}

func doUpload(ctx Context, c BaseArgs, dat io.Reader) (string, error) {
	fmt.Printf("uploading profile to %s...\n", c.Remote)
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "pprof.pb.gz")
	if err != nil {
		return "", err
	}
	_, err = io.Copy(part, dat)
	if err != nil {
		return "", err
	}
	err = writer.Close()
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.Remote.JoinPath("upload").String(), body)
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", writer.FormDataContentType())
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.ToValidUTF8(string(bts), "�"), nil
}
