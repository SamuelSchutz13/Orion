package inventory

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/SamuelSchutz13/orion-logger/internal/logs"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// DockerSource fala com a Docker Engine API pelo named pipe (sem precisar do Docker CLI).
type DockerSource struct{ cli *client.Client }

func NewDockerSource() *DockerSource {
	cli, _ := client.New(client.FromEnv) // só falha com env inválida; List reporta o erro
	return &DockerSource{cli: cli}
}

func (s *DockerSource) Name() string { return "docker" }

func (s *DockerSource) List(ctx context.Context) ([]Item, error) {
	if s.cli == nil {
		return nil, errors.New("cliente Docker inválido (verifique DOCKER_HOST)")
	}
	res, err := s.cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		if client.IsErrConnectionFailed(err) {
			return nil, errors.New("Docker não está rodando")
		}
		return nil, err
	}
	items := make([]Item, 0, len(res.Items))
	for _, c := range res.Items {
		items = append(items, containerItem(c))
	}
	return items, nil
}

func containerItem(c container.Summary) Item {
	name := c.ID
	if len(name) > 12 {
		name = name[:12]
	}
	if len(c.Names) > 0 {
		name = strings.TrimPrefix(c.Names[0], "/")
	}
	ports := []int{}
	for _, p := range c.Ports {
		if p.PublicPort != 0 && !slices.Contains(ports, int(p.PublicPort)) {
			ports = append(ports, int(p.PublicPort))
		}
	}
	slices.Sort(ports)
	it := Item{
		ID: "docker:" + c.ID, Kind: KindDocker, Name: name, Image: c.Image, Ports: ports, HasLogs: true,
		Detail: map[string]string{"state": string(c.State), "status": c.Status},
	}
	switch c.State {
	case container.StateRunning:
		it.Status, it.Actions = StatusOK, []string{"stop", "restart"}
	case container.StateRestarting, container.StatePaused:
		it.Status, it.Actions = StatusWarn, []string{"stop", "restart"}
	default:
		it.Status, it.Actions = StatusDown, []string{"start"}
	}
	return it
}

func (s *DockerSource) Act(ctx context.Context, id, action string) error {
	var err error
	switch action {
	case "start":
		_, err = s.cli.ContainerStart(ctx, id, client.ContainerStartOptions{})
	case "stop":
		_, err = s.cli.ContainerStop(ctx, id, client.ContainerStopOptions{})
	case "restart":
		_, err = s.cli.ContainerRestart(ctx, id, client.ContainerRestartOptions{})
	default:
		return fmt.Errorf("ação %q inválida para container", action)
	}
	return err
}

// StreamLogs equivale a `docker logs -f --tail 500`; bloqueia até o ctx acabar ou o stream fechar.
func (s *DockerSource) StreamLogs(ctx context.Context, id string, emit logs.Emit) error {
	insp, err := s.cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		return err
	}
	rc, err := s.cli.ContainerLogs(ctx, id, client.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true, Tail: "500"})
	if err != nil {
		return err
	}
	defer rc.Close()
	out, errw := logs.LineWriter("out", emit), logs.LineWriter("err", emit)
	defer out.Close()
	defer errw.Close()
	if insp.Container.Config != nil && insp.Container.Config.Tty {
		_, err = io.Copy(out, rc) // com TTY o stream não é multiplexado
	} else {
		_, err = stdcopy.StdCopy(out, errw, rc)
	}
	return err
}
