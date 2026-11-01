package server

import (
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/SamuelSchutz13/orion-logger/internal/actions"
	"github.com/SamuelSchutz13/orion-logger/internal/inventory"
	"github.com/gin-gonic/gin"
)

// procTarget extrai PID e horário de criação de um item proc: ou run: ainda rodando.
func procTarget(d Deps, it inventory.Item) (int32, int64, bool) {
	switch it.Kind {
	case inventory.KindProc:
		parts := strings.Split(it.ID, ":")
		if len(parts) != 3 {
			return 0, 0, false
		}
		pid, err1 := strconv.ParseInt(parts[1], 10, 32)
		ct, err2 := strconv.ParseInt(parts[2], 10, 64)
		return int32(pid), ct, err1 == nil && err2 == nil
	case inventory.KindRun:
		if d.Runs != nil {
			return d.Runs.proc(it.ID)
		}
	}
	return 0, 0, false
}

func previewAction(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		it, ok := d.Collector.Find(c.Query("id"))
		if !ok {
			c.JSON(404, gin.H{"error": "item não encontrado, atualize a lista"})
			return
		}
		if pid, ct, ok := procTarget(d, it); ok {
			info, err := actions.Preview(pid, ct)
			if err != nil {
				actionError(c, err)
				return
			}
			c.JSON(200, info)
			return
		}
		c.JSON(200, actions.PreviewInfo{Name: it.Name, PID: int32(it.PID)})
	}
}

func doAction(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct{ ID, Action string }
		if c.ShouldBindJSON(&body) != nil {
			c.JSON(400, gin.H{"error": "JSON inválido"})
			return
		}
		it, ok := d.Collector.Find(body.ID)
		if !ok {
			c.JSON(404, gin.H{"error": "item não encontrado, atualize a lista"})
			return
		}
		if !slices.Contains(it.Actions, body.Action) {
			c.JSON(400, gin.H{"error": "ação " + body.Action + " não vale para este item"})
			return
		}
		ctx := c.Request.Context()
		var err error
		switch {
		case body.Action == "kill":
			pid, ct, ok := procTarget(d, it)
			if !ok {
				err = actions.ErrChanged
				break
			}
			err = actions.KillTree(pid, ct)
		case it.Kind == inventory.KindRun && body.Action == "dismiss" && d.Runs != nil:
			d.Runs.Dismiss(it.ID)
		case it.Kind == inventory.KindLogFile && body.Action == "dismiss" && d.LogFiles != nil:
			err = d.LogFiles.Remove(strings.TrimPrefix(it.ID, "file:"))
		case it.Kind == inventory.KindDocker && d.Docker != nil:
			err = d.Docker.Act(ctx, strings.TrimPrefix(it.ID, "docker:"), body.Action)
		case it.Kind == inventory.KindPod && body.Action == "delete" && d.Kube != nil:
			err = d.Kube.DeletePod(ctx, it.Namespace, it.Name)
		default:
			err = errors.New("ação indisponível")
		}
		if err != nil {
			actionError(c, err)
			return
		}
		refresh(d)
		c.Status(204)
	}
}

func actionError(c *gin.Context, err error) {
	code := 500
	switch {
	case errors.Is(err, actions.ErrChanged):
		code = 409
	case errors.Is(err, actions.ErrDenied):
		code = 403
	}
	c.JSON(code, gin.H{"error": err.Error()})
}
