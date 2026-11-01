// orion: painel local de portas, processos, Docker e Kubernetes.
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"time"

	"github.com/SamuelSchutz13/orion-logger/internal/inventory"
	"github.com/SamuelSchutz13/orion-logger/internal/logs"
	"github.com/SamuelSchutz13/orion-logger/internal/server"
	"github.com/SamuelSchutz13/orion-logger/internal/session"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "serve" {
		args = args[1:]
	}
	fs := flag.NewFlagSet("orion", flag.ExitOnError)
	port := fs.Int("port", 7777, "porta local (cai para uma livre se ocupada)")
	noBrowser := fs.Bool("no-browser", false, "não abrir o navegador")
	fs.Parse(args)
	if err := serve(*port, !*noBrowser); err != nil {
		fmt.Fprintln(os.Stderr, "orion:", err)
		os.Exit(1)
	}
}

func serve(port int, browser bool) error {
	s, err := session.New()
	if err != nil {
		return err
	}
	l, err := server.Listen(port)
	if err != nil {
		return err
	}
	s.Port = l.Addr().(*net.TCPAddr).Port
	if err := s.Save(); err != nil {
		return err
	}
	defer session.Remove()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	procs, docker, kube := inventory.NewProcSource(), inventory.NewDockerSource(), inventory.NewKubeSource()
	hub, files := logs.NewHub(5000), server.NewLogFiles()
	runs := server.NewRuns(hub)
	extra := func(items []inventory.Item) []inventory.Item { return append(runs.Merge(items), files.Items()...) }
	col := inventory.NewCollector(2*time.Second, extra, procs, docker, kube)
	go col.Run(ctx)
	deps := server.Deps{Collector: col, Hub: hub, Docker: docker, Kube: kube, LogFiles: files, Runs: runs}
	srv := &http.Server{Handler: server.New(s, deps)}
	go func() { <-ctx.Done(); srv.Close() }()

	url := fmt.Sprintf("http://127.0.0.1:%d/?t=%s", s.Port, s.Token)
	fmt.Println("Orion Logger rodando em", url)
	if browser {
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
	if err := srv.Serve(l); err != http.ErrServerClosed {
		return err
	}
	return nil
}
