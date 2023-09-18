package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"tuxpa.in/a/pprofweb/cli"
	"github.com/alecthomas/kong"
)

func main() {
	ctx := NewCLI()
	sctx, cn := signal.NotifyContext(context.Background(), os.Kill, syscall.SIGTERM, syscall.SIGILL)
	defer cn()
	if err := ctx.Run(cli.Context{Context: sctx}); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func NewCLI() *kong.Context {
	ctx := kong.Parse(&cli.CLI)
	return ctx
}
