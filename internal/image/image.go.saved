package image

// Build an image from a Dockerfile, using runc, without the need for a docker daemon
//
// This file contains the implementation of the build command found within the
// img tool written by the genuinetools team.  It was been modified to be used
// as a library by duat.  To see the original code please checkout
// https://github.com/genuinetools/img/blob/master/build.go

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/containerd/console"
	"github.com/containerd/containerd/namespaces"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/docker/distribution/reference"
	"github.com/genuinetools/img/client"
	"github.com/genuinetools/img/types"
	controlapi "github.com/moby/buildkit/api/services/control"
	bkclient "github.com/moby/buildkit/client"
	"github.com/moby/buildkit/identity"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/moby/buildkit/util/progress/progressui"
	"golang.org/x/sync/errgroup"

	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
	"github.com/karlmutch/stack"  // Forked copy of https://github.com/go-stack/stack
)

type BuildCommand struct {
	buildArgs      []string
	dockerfilePath string
	target         string
	tag            string

	contextDir string
	noConsole  bool
}

func NewBuildCmd(dir string, target string, tag string, buildArgs []string) (bc *BuildCommand, err errors.Error) {

	bc = &BuildCommand{
		dockerfilePath: filepath.Join(dir, "./Dockerfile"),
		buildArgs:      buildArgs,
		target:         target,
		tag:            tag,
		contextDir:     dir,
		noConsole:      true,
	}

	if len(dir) == 0 || dir == "." {
		cwd, errGo := os.Getwd()
		if errGo != nil {
			return nil, errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
		}
		bc.contextDir = cwd
		bc.dockerfilePath = filepath.Join(cwd, "./Dockerfile")
	}
	bc.dockerfilePath = filepath.Clean(bc.dockerfilePath)

	if _, errGo := os.Stat(bc.dockerfilePath); os.IsNotExist(errGo) {
		return nil, errors.Wrap(errGo).With("file", bc.dockerfilePath).With("stack", stack.Trace().TrimRuntime())
	}

	named, errGo := reference.ParseNormalizedNamed(bc.tag)
	if errGo != nil {
		return nil, errors.Wrap(errGo, "invalid image name").With("tag", bc.tag).With("stack", stack.Trace().TrimRuntime())
	}
	// This will add the latest lag if they did not provide one
	bc.tag = reference.TagNameOnly(named).String()

	return bc, nil
}

func (cmd *BuildCommand) Build(stateDir string, logger chan<- string) (err errors.Error) {

	if cmd.tag == "" {
		return errors.New("tag for the generated image is missing").With("stack", stack.Trace().TrimRuntime())
	}

	if cmd.contextDir == "" {
		return errors.New("directory for the build context is empty").With("stack", stack.Trace().TrimRuntime())
	}

	// Set the dockerfile path as the default if one was not given.
	if cmd.dockerfilePath == "" {
		p, errGo := securejoin.SecureJoin(cmd.contextDir, "Dockerfile")
		if errGo != nil {
			return errors.Wrap(errGo).With("dockerfile", p).With("dir", cmd.contextDir).With("stack", stack.Trace().TrimRuntime())
		}
		cmd.dockerfilePath = p
	}

	// Create the client.
	c, errGo := client.New(stateDir, types.AutoBackend, cmd.getLocalDirs())
	if errGo != nil {
		return errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}
	defer c.Close()

	// Create the frontend attrs.
	frontendAttrs := map[string]string{
		// We use the base for filename here becasue we already set up the local dirs which sets the path in createController.
		"filename": filepath.Base(cmd.dockerfilePath),
		"target":   cmd.target,
	}

	// Get the build args and add them to frontend attrs.
	for _, buildArg := range cmd.buildArgs {
		kv := strings.SplitN(buildArg, "=", 2)
		if len(kv) != 2 {
			return errors.New("argument had an invalid key value (k=v) format").With("arg", buildArg).With("stack", stack.Trace().TrimRuntime())
		}
		frontendAttrs["build-arg:"+kv[0]] = kv[1]
	}

	if logger != nil {
		buffer := strings.Builder{}
		buffer.WriteString("building ")
		buffer.WriteString(cmd.tag)
		buffer.WriteString("\nsetting up the rootfs... this may take a bit")

		select {
		case logger <- buffer.String():
		case <-time.After(time.Second):
		}
	}

	// Create the context.
	ctx := appcontext.Context()
	sess, sessDialer, errGo := c.Session(ctx)
	if errGo != nil {
		return errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}

	id := identity.NewID()
	ctx = session.NewContext(ctx, sess.ID())
	ctx = namespaces.WithNamespace(ctx, "buildkit")
	eg, ctx := errgroup.WithContext(ctx)

	ch := make(chan *controlapi.StatusResponse)
	eg.Go(func() error {
		return sess.Run(ctx, sessDialer)
	})

	// Solve the dockerfile.
	eg.Go(func() error {
		defer sess.Close()
		return c.Solve(ctx, &controlapi.SolveRequest{
			Ref:      id,
			Session:  sess.ID(),
			Exporter: "image",
			ExporterAttrs: map[string]string{
				"name": cmd.tag,
			},
			Frontend:      "dockerfile.v0",
			FrontendAttrs: frontendAttrs,
		}, ch)
	})

	// Capture logging style output from the builder
	eg.Go(func() error {

		output := &WriterChannel{
			c:       make(chan<- string),
			timeout: time.Duration(20 * time.Millisecond),
		}

		return showProgress(ch, cmd.noConsole, output)
	})

	if errGo := eg.Wait(); errGo != nil {
		return errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}

	if logger != nil {
		buffer := strings.Builder{}
		buffer.WriteString("successfully built ")
		buffer.WriteString(cmd.tag)
		select {
		case logger <- buffer.String():
		case <-time.After(time.Second):
		}
	}

	return nil
}

func (cmd *BuildCommand) getLocalDirs() map[string]string {
	return map[string]string{
		"context":    cmd.contextDir,
		"dockerfile": filepath.Dir(cmd.dockerfilePath),
	}
}

type WriterChannel struct {
	c       chan<- string
	timeout time.Duration
}

func (wc *WriterChannel) Write(p []byte) (n int, err error) {
	if wc != nil && wc.c != nil {
		select {
		case wc.c <- string(p):
			return
		case <-time.After(wc.timeout):
		}
	}
	fmt.Sprintln(string(p))
	return len(p), nil
}

func showProgress(ch chan *controlapi.StatusResponse, noConsole bool, sink *WriterChannel) error {
	displayCh := make(chan *bkclient.SolveStatus)
	go func() {
		for resp := range ch {
			s := bkclient.SolveStatus{}
			for _, v := range resp.Vertexes {
				s.Vertexes = append(s.Vertexes, &bkclient.Vertex{
					Digest:    v.Digest,
					Inputs:    v.Inputs,
					Name:      v.Name,
					Started:   v.Started,
					Completed: v.Completed,
					Error:     v.Error,
					Cached:    v.Cached,
				})
			}
			for _, v := range resp.Statuses {
				s.Statuses = append(s.Statuses, &bkclient.VertexStatus{
					ID:        v.ID,
					Vertex:    v.Vertex,
					Name:      v.Name,
					Total:     v.Total,
					Current:   v.Current,
					Timestamp: v.Timestamp,
					Started:   v.Started,
					Completed: v.Completed,
				})
			}
			for _, v := range resp.Logs {
				s.Logs = append(s.Logs, &bkclient.VertexLog{
					Vertex:    v.Vertex,
					Stream:    int(v.Stream),
					Data:      v.Msg,
					Timestamp: v.Timestamp,
				})
			}
			displayCh <- &s
		}
		close(displayCh)
	}()

	if sink != nil {
		return progressui.DisplaySolveStatus(context.TODO(), nil, sink, displayCh)
	}

	var c console.Console
	if !noConsole {
		if cf, err := console.ConsoleFromFile(os.Stderr); err == nil {
			c = cf
		}
	}

	return progressui.DisplaySolveStatus(context.TODO(), c, os.Stdout, displayCh)
}

func Prune(stateDir string) (err errors.Error) {
	// Create the client.
	c, errGo := client.New(stateDir, types.AutoBackend, nil)
	if err != nil {
		return errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}
	defer c.Close()

	// Create the context.
	ctx := session.NewContext(appcontext.Context(), identity.NewID())
	ctx = namespaces.WithNamespace(ctx, "buildkit")

	if _, errGo := c.Prune(ctx); errGo != nil {
		return errors.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}
	return nil
}
