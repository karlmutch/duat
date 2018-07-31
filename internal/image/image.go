package duat

// This file contains the implementation of the build command found within the
// img tool written by the genuinetools team.  It was been modified to be used
// as a library by duat.  To see the original code please checkout
// https://github.com/genuinetools/img/blob/master/build.go

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containerd/console"
	"github.com/containerd/containerd/namespaces"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/docker/distribution/reference"
	"github.com/genuinetools/img/client"
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

const buildHelp = `Build an image from a Dockerfile, using runc, without the need for a docker daemon.`

func (cmd *BuildCommand) Register(fs *flag.FlagSet) {
	fs.StringVar(&cmd.dockerfilePath, "f", "", "Name of the Dockerfile (Default is 'PATH/Dockerfile')")
	fs.StringVar(&cmd.tag, "t", "", "Name and optionally a tag in the 'name:tag' format")
	fs.StringVar(&cmd.target, "target", "", "Set the target build stage to build")
	fs.Var(&cmd.buildArgs, "build-arg", "Set build-time variables")
	fs.BoolVar(&cmd.noConsole, "no-console", false, "Use non-console progress UI")
}

type BuildCommand struct {
	buildArgs      stringSlice
	dockerfilePath string
	target         string
	tag            string

	contextDir string
	noConsole  bool
}

func NewBuildCmd() (bc *BuildCommand) {
	bc = &BuildCommand{}
}

func (cmd *BuildCommand) Run(args []string) (err error) {
	if len(args) < 1 {
		return errors.New("no dockerfile location was passed").With("stack", stack.Trace().TrimRuntime())
	}

	if cmd.tag == "" {
		return errors.New("tag for the generated image is missing").With("stack", stack.Trace().TrimRuntime())
	}

	// Get the specified context.
	cmd.contextDir = args[0]

	if cmd.contextDir == "" {
		return errors.New("directory for the build context is empty").With("stack", stack.Trace().TrimRuntime())
	}

	// Parse the image name and tag.
	named, errGo := reference.ParseNormalizedNamed(cmd.tag)
	if errGo != nil {
		return errors.Wrap(errGo, fmt.Sprintf("parsing image name %q failed", cmd.tag)).With("tag", cmd.tag).With("stack", stack.Trace().TrimRuntime())
	}
	// Add the latest lag if they did not provide one.
	cmd.tag = reference.TagNameOnly(named).String()

	// Set the dockerfile path as the default if one was not given.
	if cmd.dockerfilePath == "" {
		cmd.dockerfilePath, errGo = securejoin.SecureJoin(cmd.contextDir, defaultDockerfileName)
		if err != nil {
			return errors.Wrap(errGo).With("dockerfile", defaultDockerfileName).With("dir", cmd.contextDir).With("stack", stack.Trace().TrimRuntime())
		}
	}

	// Create the client.
	c, errGo := client.New(stateDir, backend, cmd.getLocalDirs())
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

	fmt.Printf("Building %s\n", cmd.tag)
	fmt.Println("Setting up the rootfs... this may take a bit.")

	// Create the context.
	ctx := appcontext.Context()
	sess, sessDialer, err := c.Session(ctx)
	if err != nil {
		return err
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
	eg.Go(func() error {
		return showProgress(ch, cmd.noConsole)
	})
	if err := eg.Wait(); err != nil {
		return err
	}
	fmt.Printf("Successfully built %s\n", cmd.tag)

	return nil
}

func (cmd *BuildCommand) getLocalDirs() map[string]string {
	return map[string]string{
		"context":    cmd.contextDir,
		"dockerfile": filepath.Dir(cmd.dockerfilePath),
	}
}

func showProgress(ch chan *controlapi.StatusResponse, noConsole bool) error {
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
	var c console.Console
	if !noConsole {
		if cf, err := console.ConsoleFromFile(os.Stderr); err == nil {
			c = cf
		}
	}
	return progressui.DisplaySolveStatus(context.TODO(), c, os.Stdout, displayCh)
}
