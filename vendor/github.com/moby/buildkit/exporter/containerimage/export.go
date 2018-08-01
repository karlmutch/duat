package containerimage

import (
	"context"
	"strconv"
	"time"

	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/moby/buildkit/cache"
	"github.com/moby/buildkit/exporter"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/util/push"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	keyImageName        = "name"
	keyPush             = "push"
	keyInsecure         = "registry.insecure"
	exporterImageConfig = "containerimage.config"
	ociTypes            = "oci-mediatypes"
)

type Opt struct {
	SessionManager *session.Manager
	ImageWriter    *ImageWriter
	Images         images.Store
}

type imageExporter struct {
	opt Opt
}

// New returns a new containerimage exporter instance that supports exporting
// to an image store and pushing the image to registry.
// This exporter supports following values in returned kv map:
// - containerimage.digest - The digest of the root manifest for the image.
func New(opt Opt) (exporter.Exporter, error) {
	im := &imageExporter{opt: opt}
	return im, nil
}

func (e *imageExporter) Resolve(ctx context.Context, opt map[string]string) (exporter.ExporterInstance, error) {
	i := &imageExporterInstance{imageExporter: e}
	for k, v := range opt {
		switch k {
		case keyImageName:
			i.targetName = v
		case keyPush:
			if v == "" {
				i.push = true
				continue
			}
			b, err := strconv.ParseBool(v)
			if err != nil {
				return nil, errors.Wrapf(err, "non-bool value specified for %s", k)
			}
			i.push = b
		case keyInsecure:
			if v == "" {
				i.insecure = true
				continue
			}
			b, err := strconv.ParseBool(v)
			if err != nil {
				return nil, errors.Wrapf(err, "non-bool value specified for %s", k)
			}
			i.insecure = b
		case exporterImageConfig:
			i.config = []byte(v)
		case ociTypes:
			if v == "" {
				i.ociTypes = true
				continue
			}
			b, err := strconv.ParseBool(v)
			if err != nil {
				return nil, errors.Wrapf(err, "non-bool value specified for %s", k)
			}
			i.ociTypes = b
		default:
			logrus.Warnf("image exporter: unknown option %s", k)
		}
	}
	return i, nil
}

type imageExporterInstance struct {
	*imageExporter
	targetName string
	push       bool
	insecure   bool
	ociTypes   bool
	config     []byte
}

func (e *imageExporterInstance) Name() string {
	return "exporting to image"
}

func (e *imageExporterInstance) Export(ctx context.Context, ref cache.ImmutableRef, opt map[string][]byte) (map[string]string, error) {
	if config, ok := opt[exporterImageConfig]; ok {
		e.config = config
	}
	desc, err := e.opt.ImageWriter.Commit(ctx, ref, e.config, e.ociTypes)
	if err != nil {
		return nil, err
	}

	defer func() {
		e.opt.ImageWriter.ContentStore().Delete(context.TODO(), desc.Digest)
	}()

	if e.targetName != "" {
		if e.opt.Images != nil {
			tagDone := oneOffProgress(ctx, "naming to "+e.targetName)
			img := images.Image{
				Name:      e.targetName,
				Target:    *desc,
				CreatedAt: time.Now(),
			}

			if _, err := e.opt.Images.Update(ctx, img); err != nil {
				if !errdefs.IsNotFound(err) {
					return nil, tagDone(err)
				}

				if _, err := e.opt.Images.Create(ctx, img); err != nil {
					return nil, tagDone(err)
				}
			}
			tagDone(nil)
		}
		if e.push {
			if err := push.Push(ctx, e.opt.SessionManager, e.opt.ImageWriter.ContentStore(), desc.Digest, e.targetName, e.insecure); err != nil {
				return nil, err
			}
		}
	}

	return map[string]string{
		"containerimage.digest": desc.Digest.String(),
	}, nil
}
