# stencil

Runs `stencil`. To learn more about `stencil`, see the [duat repository](https://github.com/karlmutch/duat#stencil/).

This action is intended to be used to process templated Dockerfile to inject release and other information
and then to pass the output to the docker build command.

Use `WORKING_DIRECTORY` to specify the dockerfile directory to use. Defaults to `.`.
Use `DOCKERFILE` to name the actual Dockerfile to be used from the WORKING_DIRECTORY.
Use `DOCKERFILE_STENCIL` to give the output Dockerfile a name, this file is typically passed to a Dockerbuild in the next step.

```hcl
action "golint" {
  uses    = "karlmutch/duat/stencil"
  needs   = "previous-action"
  secrets = ["GITHUB_TOKEN"]

  env {
    WORKING_DIR = "./path/to/Dockerfile"
    DOCKERFILE = "Dockerfile"
    DOCKERFILE_STENCIL = "Dockerfile.stencil"
  }
}
```
