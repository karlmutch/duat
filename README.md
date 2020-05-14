# Developer utilities and tools (duat) Beta

Version : <repo-version>0.13.0</repo-version>

duat is a set of tools useful for automating the bootstrapping of containerized workflows.  duat includes tools for working with software artifacts such as git branches and tags, semantic versioning, and docker image delivery.  duat is a work in progress experiment in using Go, and Kubernetes to manage portions of container centric software lifecycles, helping to remove proprietary tooling, scripting, and other DSLs typically used for building, releasing, and deploying software.

This repository also includes go import packages for handling meta data and artifacts associated with software development activities.

duat is opinionated about naming of docker images, and semantic versioning.  duat provides tools and assistance specifically for go based development, but intentionally does not impose an end-to-end CI/CD automation solution.  Downstream workflow choices are determined by the user not duat.

The duat git-watch bootstrapping tool elevates the platform layer from an OS host using docker, to using Kubernetes as the platform.  This includes the ability to build docker images.

# About

This project was started as a means of experimenting with continuous integration using go as the primary implemention language also as my primary means of automatting build and release processes.  Other projects such as mage, https://magefile.org/, also do the same.  This project was started with the intention of working with go libraries primarily for handling versioning, git source control, and containerization.  Mage in contrast leverages a strategy of wrapping shell commands to achieve this.

Overtime the objective of duat has changed from being used across the entire workflow to filling in the gaps for existing containerized CI/CD solutions in relation to deployments that wish to avoid hard dependencies on vulnerable public infrastructure.

The issue of operating system configuration management for developer environments where docker is seen as a barrier to fast build, test, debug cycles due to speed is best addressed using tools such as Ansible for which I have a seperate github project, github.com/karlmutch/DevBoot.  The Dockerfiles however that a developer creates for their development projects act as a the last word in regards to the supported environments for projects.  Because of the importance of software configuration management release builds should be done using containers and reference images.

# The name duat

Duat was the name given by the ancient egyptians to the underworld.  As the sun set each day and travelled through the underworld being regenerated it would cast light on to the souls nearby bringing them to life for a period of time as it passed by.  While passing through the duat the hope is that these tools help to light each of the hours in your build pipeline.

# Motivating use-cases

A users self hosted Kubernetes hosted CI/CD pipeline is used to remove the need for external dependencies on service providers such as Travis or Quay.io.

A user wishes to develop and build software producing distinct tagged docker images for every build and, clean up built docker images of previous development versions as development progresses.

A user wishes to deliver software packaged using docker images to an AWS ECR image repository.

A user wishes to deploy containerized software into an Istio, or other Kubernetes based service mesh.

Many existing cloud based platforms exist today to address requirements such as the above.  This has led to divided islands of functionality, such as Travis, that require integration with each other so that credentials and other artifacts required by workflow automation is shared between these platform. Costly and fragile integration falls to the developer and users which is time consuming and complex.  duat builds upon the observation that many developers are already operating in a cloud based environmenti, or other Kubernetes capable offering and what is needed is a simple set of tools, if a set of simplifing assumptions is made for addressing the above use cases, especially if you already have containerized builds, and tests.

For example the following workflow might be used to compile duat itself:

```shell
go run ./build.go -r cmd > /tmp/compiled.lst
# test is run and passes
go run ./cmd/github-release/github-release.go `cat /tmp/compiled.lst`
```

# Conventions assumptions

The tools and packages within this project rely on a couple of conventions and assumptions.

1. git

    git is the primary source code management tool

    git tags are semver 2.0 compliant

2. release targets

    can be containers stored using docker registries

    can be github releases

3. semantic versioning

    using prefix characters are not semver compliant and are not used

    semver pre-release versions are sortable, are ordered, are reversable for a time stamp, and obey DNS character rules

4. containerization and Kubernetes

    builds are performed using containers

    container orchestration for pielines and image creation is done using Kubernetes for both host and workstation scenarios

Deployment concerns for the pipeline support is addressed on hosts through the use of standard Kubernetes distributions.  

OSX based laptops and PC Workstations can use the Ubuntu distribution deployed using VirtualBox, WSL on Windows 10 Professional, or using the multipass project for OSX and Windows 10 Professional.  All of these allow the microk8s Kubernetes distribution to run, differentiated from minikube via support for GPUs, an Istio distribution built in, along with a image registry and other tools.


# duat configuration

The duat tools use the context of the current directory and environment variables to use git, AWS, and docker.

git context is determined by locating the .git directory working from the current working directory and successively through the directories parents until found.  Once the .git directory is found the repository URL, repo name, branch names and tags will all be associated with that directory.  Tools that use git also over a '-git' option to manually specify where the git context comes from.  Current tools do not perform git push/pull operations so are not impacted by the use of the GITHUB_TOKEN environment variable, when/if these features are added then this environment variable will be used.

AWS context is determined using the AWS environment variables, and AWS credentials file as documented by items 2, and 3 of AWS configuration settings and precedence documented at, https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html#config-settings-and-precedence.

When using AWS ECR to store containers the default container registry will be used for your AWS account and containers labeled and named using their github repository and directory name for the compiled component in the case of Go code.  Python is yet to be determined and not set by the tools.

Kubernetes context does not impact the current tool set offered other than for container naming and AWS ECR hosting.

Because the current shell context sets the stage for using these external platforms no configuration of duat is currently needed as it is automatic.

# Versioning

duat generates and executes builds using containers with github repository names, and tags artifacts using the branch name with versions.

By default the version number of the current repo is stored within the README.md file in the git root directory.  Any other file can be used, for example VERSION at the developers choice using the -f option.

## Version rules

duat generates and software under management using the semantic versioning 2.0 and uses the additional semver pre-release label.  For example:

```
karlmutch/no-code/noserver:0.1.0-89-bzip2-aaaagjecpnf
```

represents a docker image from the github.com/karlmutch/no-code repository, and the noserver component, that is a pre-release of 0.1.0 and was generated from the 89\_bzip2, or 89-bzip2 branch with the pre-release timestamp of aaaagjecpnf.

The git repository name is obtained by the git package

The version portion of the semver is wrangled by the semver package using the README.md file as the authortative source.

The pre-release portion is obtained from git using the branch name, and the trailing portion 'aaaagjecpnf' is a Base 24 encoding, modified to allow sorting is date time order of the string, pre-release stamped within the README.md file.

## Versioning Workflow

A typical workflow for versioning is to start by using semver to increment the version based upon the major, minor, patch changes and then apply the new version of the existing README.md file.  If you are doing development then the first step is to use github to generate or identify a ticket and then to create a branch using the ticket identifier as the branch name with a description. Then, having done this a git checkout would be used to obtain the branch and then the semver tools use to increment the version string, typically 'semver patch' and follow it up with setting the pre-release version 'semver pre' to add the pre-release tag.  As changes are made and new compiles are successful the 'semver pre' command can continue to be used between succesful compiles if needed to generate new versions within docker.  This is useful when doing testing within an existing kubernetes cluster where upgrades of the software are done to test services.

Once development has completed, a working pre-release version of the build is identified, and tested the developer would use the semver command to clear the prerelease indicator and then tag the release image, followed by a push to the remote docker repository that houses released artifacts.  This would then be followed by releasing git binary artifacts to github, if needed. For example:

```
$ semver extract
0.0.1
$ git checkout 00_new_feature
$ semver patch
0.0.2
$ semver pre
0.0.2-00-new-feature-aaaagjecrgf
```
Development is done and testing carried out
A decision is made to release and PR is approved

```
$ image-promote
0.0.2
$ github-release
0.0.2
```

# Python environments

# Go environments

The motivation for this project is a requirement to supply software that can be employed when automating build and delivery activities using Go based projects.

Using Go to automate CI/CD style activities is an attempt to address the following:

1. hosted CI/CD solutions create a needless separation with new styles of cloud based development and containerization

2. using bash and vendor specific CI/CD DSLs to automate is disruptive and hard to debug

The individual tools this projects provides can be run in multiple ways:

1. as standalone utilities to do very discret actions when invoked by bash scripts or on the command line.

2. as packages inside go source files, the source files being executed in the same manner as shell scripts.

3. As builds that are triggered by pushed commits in a shared git repository.

The go packages this project implements are intended to be employed when using Go as a scripting language, as an alternative to bash scripts.  Information about using Go as a shell scripting language can be found at, https://blog.cloudflare.com/using-go-as-a-scripting-language-in-linux/.

The general idea is to produce both libraries for development artifact handling and also tools that can be invoked to perform administrative functions related to software release and build activities.

# Installation using Github binaries

duat has regular releases of the stable head of the git repo.  The release consists at this time of precompiled Linux x86\_64 binaries that can be found at https://github.com/karlmutch/duat/releases.

# Installation using go get

duat is go gettable with the command tools compiled using the go tools.  The following commands will suffice for most Go environments.

```
go get github.com/karlmutch/duat
go install github.com/karlmutch/duat/cmd/semver
go install github.com/karlmutch/duat/cmd/github-release
go install github.com/karlmutch/duat/cmd/stencil
go install github.com/karlmutch/duat/cmd/git-watch
```

# Building duat from source

## Prerequisites

```
go get github.com/erning/gorun
sudo cp $GOPATH/bin/gorun /usr/local/bin/gorun
echo ':golang:E::go::/usr/local/bin/gorun:OC' | sudo tee /proc/sys/fs/binfmt_misc/register
```

### Docker based Release builds

Using build.sh

If you have the environment variable GITHUB\_TOKEN defined then this will be imported into the build container and used for perform releases of the software at the semver for the project.

### Development builds

Using build.go

```
go run ./build.go -r cmd > /tmp/compiled.lst
go test -v ./...
```

### Perfoming a release

```
semver [patch|minor|major]
go test -v ./...
go run ./build.go -r cmd > /tmp/compiled.lst
cat /tmp/compiled.lst | go run ./cmd/github-release/github-release.go -
```

# duat utilities and tools

## semver
A version handling tool for storing and manipuating a semantic version (semver) within files.  The semantic version is expressed using an HTML formatted repo-version tag.

This tool also has the ability to generate semver compliant version strings using git branch names when the pre command is used.

This tool applies the Semantic Versioning 2.0 spec to version strings in files, https://semver.org/.  When the pre command is used the version bumping will append to the version string an increasing pre-release string that can be used to sort the versions creating precedence for versions when they are used with containers or other assets.

### Installation

semver can installed using the following command:

```shell
$ go get -u karlmutch/duat/cmd/semver
```

### Basic usage

semver by default will read your README.md file and will examine it for HTML markup embeeded in the file <doc-opt><code>&lt;repo-version&gt;[semver]&lt;/repo-version&gt;</code></doc-opt>.  The text within the tag will be parsed and validated as being valid semver, if this fails the command will exit.  Once parsed the options specified on the semver command line will be used to morph the version and written back into the file.

semver can also be used with the apply option to modify files based upon the version within an authorative file.  When this option is used not changes are made to the existing input file.  This command is only for propagating an existing version to other files.

semver will output to stdout the new version number, except for the apply command where you will get the current version applies to the target-file list.

The command has the following usage:

<doc-opt><code>
Semantic Version tool

Usage:

  semver [major | major | minor | pre | extract | apply] [-f=<input-file>] [-t=[&lt;target-file&gt;,...]]

Options:
  -h --help              Show this message.<p>
  -version               Show the version of this software.<p>
  -git string            The top level of the git repo to be used for the dev version (default ".")
  -f=&lt;input-file&gt;        A file containing an HTML repo-version tag to be morped or queried [default: README.md]<p>
  -t=&lt;target-file&gt;,...   A comma seperated list of files that will be examined for version tags and modified based upon the input-file version<p>
</code></doc-opt>

## github-release

This tool can be used to push binaries and other files upto github using the current version as the tag for the release.

## stencil

This tool is a general purpose template processing utility that reads template files and substitutes values from the software integration envirionment and runs functions specified within the template.

```shell
stencil -input  example/artifact/Dockerfile
```

stencil support go templating for substitution of variables inside the input file.  Variables are added for duat specifically including:

```
{{.duat.version}}
{{.duat.module}}
{{.duat.gitTag}}
{{.duat.gitHash}}
{{.duat.gitBranch}}
{{.duat.gitURL}}
{{.duat.gitDir}}
{{.duat.awsecr}}
```

Templates also support functions from masterminds.github.io/sprig.  Please refer to that github website for more information.

## git-watch

The primary use case for git-watch is to be able to build CI/CD docker images from git repositories when commits occur.  git-watch meets an unaddressed need for an ad-hoc git client that can produce CI/CD docker images that can then feed into container centric CI/CD pipelines, all hosted within a single node Kubernetes deployment.

This document describes by example the git-watch tool using a combination of git, docker registries, and finally keel.sh for downstream CI.

### audience

The primary audience for this style of CI/CD bootstrapping includes individual, or small teams of developers with shared and/or limited resources who wish to implement CI/CD pipelines but for which the price of entry for large scale clusters and infrastructure is too high.

This tool is useful for first capturing a git cloned repository into a docker image, pushing this into an image registry and then triggering down stream CI/CD operations.  It uses polling in order that publically accessible hosts are not needed and the costs of handling CI/CD pipelines remains minimal.  This tool is designed to place private CI/CD pipelines into the hands of smaller teams sensitive to the costs of using a SaaS solution such as Travis, and tools such as Jenkins that require a full time engineer role.

### introduction

The git-watch tool will poll a github repository and when commits are observed, clones the github repo into a Kubernetes volume, and then dispatch a Kubernetes job, for example an Uber Mikasu based docker image build.

git-watch is intended to run as the primordial step front-ending a downstream pipeline.  The git commit and push to the git origin repository acts as the trigger for git-watch to begin the process of packaging the source code at a specific commit into a Kubernetes volume.  git-watch will then take the packaged volume and run it against a Kubernetes batch Job for your choice.  Ubers Mikasu is used by the authors as the Job for processing the packaged code.  Mikasu is used to produce a docker image containing the source code and the results of the docker build using a nominated Dockerfile within the packaged volume suitable for CI/CD actions.  Mikasu will then push the docker image to a registry of the users choice as supported by Mikasu.

The trigger to the downstream pipeline is a combination of git-watch and Uber Mikasu pushing a docker image to an image registry for example docker hub, or Amazon ECR.  The use of image registries is common for several CI/CD platforms as triggers.  For the minimalist case the author makes use of https://keel.sh/ to monitor for the artifacts produced by the tools under discussion to trigger actions related to the builds, test, release lifecycle of choice.

### github

git-watch can be configured to watch git repositories using the git clone url, and optionally can be configured to watch specific branches. In order to have git-watch run continuously it can be used in combination with a Kubernetes Deployment and a containerized version of this application.

The --github-token option is used by the watcher to access any configured repositories.  Having an environment variable GITHUB\_TOKEN is also supported.

The --state-persistence-dir option is used to specify where the files that track the last seen commit ID for the github repositories is kept.

The repositories are specified as arguments to the command.  Each argument represents a git repository URL and can be suffixed with a caret character, '^', and the branch name to further specify the repository to be watched.

The --job-template option is used to specify a template following the golang/Kubernetes style that will be used to initiate jobs as commits are detected.  An exmaple of a template is provided in the duat code repository called 'ci\_containerize.yaml'.  This Kubernetes Job specification uses the Makisu container build image from Uber to read a Dockerfile from the code base and deploy an image containing all of the source code associated with the commit.  Mikasu will then push that image to a 3rd party image repository using a Kubernetes secret populated by the user.

### registry

In order to perform builds with stable code throughout the process the Mikasu job template builds a Docker image using the default Dockerfile inside the code repository being watched.  After the build image has obtained the needed compilation tooling and source code to be useful during CI/CD operations it is pushed to a registry for access by the downstream CI/CD pipeline.  The Mikasu builder uses Kubernetes secrets to retrieve the user name and password for the registry credentials.  The user is responsible for definition of the Registry environment variable to hold the credentials.  When the Job in which the Mikasu container is run the environment variable will be substituted into the Job template.

Before using the registry setting you should copy registry-template.yaml to registry.yaml, and modify the contents so that they contain your docker user name and password.  You can then apply the secrets to your environment variables using commands such as the following:

```
export Registry=`cat registry.yaml`
```

When the git-watch command is run environment variables set by the user can be substituted into your job template, the above example uses the Registry env variable to do this.

### Kubernetes microk8s

In order to make use of Kubernetes a KUBECONFIG environment variable should be set that contains the configuration items needed to access your cluster.  When using microk8s the configuration can be extracted from the microk8s tool as follows:

```
microk8s.kubectl config view --raw > $HOME/.kube/microk8s.config
export KUBECONFIG=$HOME/.kube/microk8s.config
```

The documentation in the microk8s README.md, on the github page, is very useful and a recommended read for users using a laptop or workstation style configuration.

microk8s uses a docker registry that it self deploys and which can be referenced inside the job template using the RegistryPort, and RegistryIP variables.  In order to use the insecure microk8s registry take the registry\_local.yaml file and define a Registry environment variable to hold it, then start the watcher using the example ci\_containerize\_local.yaml file as your job template.

```
export Registry=`cat registry_local.yaml`
git-watch --job-template ../../ci_containerize_local.yaml https://github.com/karlmutch/duat.git^feature/102_microk8s_registry
```

Generated images will be accessible using the host via the localhost:32000 microk8s registry proxy and can be pulled from the cluster to the local docker registry or can be left in-situ for a CI pipeline to pull and process via a tool such as https://keel.sh.

The following example shows the result of a commit triggering a build after the git-watch tool has been started on the command line:

```
$ export KUBECONFIG=~/.kube/microk8s.config
$ export KUBE_CONFIG=~/.kube/microk8s.config
$ microk8s.kubectl config view --raw > $KUBECONFIG
$ cat ../../registry_local.yaml   
localhost:
  .*:
    security:
      tls:
        client:
          disabled: true
$ export Registry=`cat ../../registry_local.yaml`
$ git-watch -v --ignore-aws-errors --job-template ../../ci_containerize_local.yaml https://github.com/karlmutch/duat.git^feature/102_microk8s_registry
16:05:42.953783 INF git-watch task update id: 3b8e99c8-aed8-40d0-8ccd-82c8db391ff5 text: volume update namespace: gw-0-11-1-feature-102-microk8s-registry-aaaagjfuypo volume: 3b8e99c8-aed8-40d0-8ccd-82c8db391ff5 phase: (v1.PersistentVolumeClaimPhase) (len=5) "Bound"
16:05:47.257452 INF git-watch task update id: 3b8e99c8-aed8-40d0-8ccd-82c8db391ff5 text: pod update phase: Pending id: 3b8e99c8-aed8-40d0-8ccd-82c8db391ff5 namespace: gw-0-11-1-feature-102-microk8s-registry-aaaagjfuypo
16:05:48.160613 INF git-watch task update id: 3b8e99c8-aed8-40d0-8ccd-82c8db391ff5 text: pod update id: 3b8e99c8-aed8-40d0-8ccd-82c8db391ff5 namespace: gw-0-11-1-feature-102-microk8s-registry-aaaagjfuypo phase: Running
16:09:52.460762 INF git-watch task update id: 3b8e99c8-aed8-40d0-8ccd-82c8db391ff5 text: pod update id: 3b8e99c8-aed8-40d0-8ccd-82c8db391ff5 namespace: gw-0-11-1-feature-102-microk8s-registry-aaaagjfuypo phase: Succeeded
16:09:52.462364 INF git-watch task update id: 3b8e99c8-aed8-40d0-8ccd-82c8db391ff5 text: running id: 3b8e99c8-aed8-40d0-8ccd-82c8db391ff5 namespace: gw-0-11-1-feature-102-microk8s-registry-aaaagjfuypo dir: /tmp/git-watcher/Eu8nD7tNm0qcxSRTojd8518tk5npRG6bn2qgQeUqplE
16:09:52.465553 INF git-watch task update id: 3b8e99c8-aed8-40d0-8ccd-82c8db391ff5 text: pod namespace: gw-0-11-1-feature-102-microk8s-registry-aaaagjfuypo node_name: awsdev pod_name: copy-pod
16:09:52.465606 INF git-watch task update id: 3b8e99c8-aed8-40d0-8ccd-82c8db391ff5 text: pod namespace: gw-0-11-1-feature-102-microk8s-registry-aaaagjfuypo node_name: awsdev pod_name: imagebuilder-k7xhn
16:09:52.465644 INF git-watch task update id: 3b8e99c8-aed8-40d0-8ccd-82c8db391ff5 text: volume namespace: gw-0-11-1-feature-102-microk8s-registry-aaaagjfuypo volume_name: 3b8e99c8-aed8-40d0-8ccd-82c8db391ff5 capacity: 10Gi
16:09:52.465830 INF git-watch task completed id: 3b8e99c8-aed8-40d0-8ccd-82c8db391ff5 dir: /tmp/git-watcher/Eu8nD7tNm0qcxSRTojd8518tk5npRG6bn2qgQeUqplE namespace: gw-0-11-1-feature-102-microk8s-registry-aaaagjfuypo
```

If we were to observe the running job from another terminal, and once complete were to pull the resulting source code image into our local docker registry we would see the following:

```
$ semver pre
0.11.1-feature-102-microk8s-registry-aaaagjfuypo
$ git commit -a -m "Bump semantic version to trigger a build via a commit"
$ git push
# Allow the build to trigger itself based on our pushed commit to the github repository
$ sleep 30
$ microk8s.kubectl get ns
NAME                                                  STATUS   AGE
container-registry                                    Active   23h
default                                               Active   23h
gw-0-11-1-feature-102-microk8s-registry-aaaagjfuypo   Active   23h
kube-node-lease                                       Active   23h
kube-public                                           Active   23h
kube-system                                           Active   23h
$ kubectl --namespace=gw-0-11-1-feature-102-microk8s-registry-aaaagjfuypo get pods                  
NAME                 READY   STATUS      RESTARTS   AGE
copy-pod             1/1     Running     0          23h
imagebuilder-k7xhn   1/1     Running     0          23h
$ kubectl --namespace=gw-0-11-1-feature-102-microk8s-registry-aaaagjfuypo logs -f imagebuilder-k7xhn
...
{"level":"info","ts":1555024191.5788004,"msg":"Stored cacheID mapping to KVStore: 960ee5ee => MAKISU_CACHE_EMPTY"}
{"level":"info","ts":1555024191.5788994,"msg":"Stored cacheID mapping to KVStore: 8c70ac78 => MAKISU_CACHE_EMPTY"}
{"level":"info","ts":1555024191.6197424,"msg":"Computed total image size 431108731","total_image_size":431108731}
{"level":"info","ts":1555024191.6197767,"msg":"Successfully built image karlmutch/duat:0.11.1-feature-102-microk8s-registry-aaaagjfuypo"}
{"level":"info","ts":1555024191.619851,"msg":"* Started pushing image 10.1.1.32:5000/karlmutch/duat:0.11.1-feature-102-microk8s-registry-aaaagjfuypo"}
{"level":"info","ts":1555024191.6351519,"msg":"* Image 10.1.1.32:5000/karlmutch/duat:0.11.1-feature-102-microk8s-registry-aaaagjfuypo already exists, overwriting"}
{"level":"info","ts":1555024191.6383321,"msg":"* Skipped pushing existing layer karlmutch/duat:sha256:119c7358fbfc2897ed63529451df83614c694a8abbd9e960045c1b0b2dc8a4a1"}
{"level":"info","ts":1555024191.6391516,"msg":"* Skipped pushing existing layer karlmutch/duat:sha256:d18d76a881a47e51f4210b97ebeda458767aa6a493b244b4b40bfe0b1ddd2c42"}
{"level":"info","ts":1555024191.6396852,"msg":"* Skipped pushing existing layer karlmutch/duat:sha256:34667c7e4631207d64c99e798aafe8ecaedcbda89fb9166203525235cc4d72b9"}
{"level":"info","ts":1555024191.641192,"msg":"* Skipped pushing existing layer karlmutch/duat:sha256:2aaf13f3eff07aa25f73813096bd588e6408b514288651402aa3d0357509be7a"}
{"level":"info","ts":1555024191.6417122,"msg":"* Skipped pushing existing layer karlmutch/duat:sha256:914e2320d76b65434df88951e6d9736cfec1ea22cfce48bfb0945ed5a5e99639"}
{"level":"info","ts":1555024191.6418197,"msg":"* Skipped pushing existing layer karlmutch/duat:sha256:5c75f60bd4e0546a26ea7d6eec32835b93c41d994291f57815bc83673f099e5c"}
{"level":"info","ts":1555024191.6461737,"msg":"* Skipped pushing existing layer karlmutch/duat:sha256:d273e0d8cc2120c145e615ddb78c627049cb15a03a5dcb10fdff9e1b07cee633"}
{"level":"info","ts":1555024191.6467197,"msg":"* Skipped pushing existing layer karlmutch/duat:sha256:5aa155938eb06162dc1e8387c80b186b0032364fdd5733da78defbb81fb56b4f"}
{"level":"info","ts":1555024191.7139697,"msg":"* Started pushing image config sha256:8370897394f1bbf6a0fbad488698364796b3f897625f16b23a92aa25a13227b7"}
{"level":"info","ts":1555024191.733266,"msg":"* Finished pushing image config sha256:8370897394f1bbf6a0fbad488698364796b3f897625f16b23a92aa25a13227b7"}
{"level":"info","ts":1555024191.7486677,"msg":"* Finished pushing image 10.1.1.32:5000/karlmutch/duat:0.11.1-feature-102-microk8s-registry-aaaagjfuypo in 128.789167ms"}
{"level":"info","ts":1555024191.7486932,"msg":"Successfully pushed 10.1.1.32:5000/karlmutch/duat:0.11.1-feature-102-microk8s-registry-aaaagjfuypo to 10.1.1.32:5000"}
{"level":"info","ts":1555024191.7487,"msg":"Finished building karlmutch/duat:0.11.1-feature-102-microk8s-registry-aaaagjfuypo"}
```

Now that the image has been pushed into the registry of the cluster it can be pulled into an external registry using port 32000 on the host that is running the microk8s cluster and used.

```
$ docker pull localhost:32000/karlmutch/duat:0.11.1-feature-102-microk8s-registry-aaaagjfuypo
0.11.1-feature-102-microk8s-registry-aaaagjfuypo: Pulling from karlmutch/duat
34667c7e4631: Already exists 
d18d76a881a4: Already exists 
119c7358fbfc: Already exists 
2aaf13f3eff0: Already exists 
5c75f60bd4e0: Pull complete 
914e2320d76b: Pull complete 
d273e0d8cc21: Pull complete 
5aa155938eb0: Pull complete 
Digest: sha256:c0f0070555204d528709f1c3a25f44e56950d9dc9b1be5a2bcb6ac73bab26ee1
Status: Downloaded newer image for localhost:32000/karlmutch/duat:0.11.1-feature-102-microk8s-registry-aaaagjfuypo
```

If you have deployed a keel based pipeline then you will be able to have it triggered off the internal registry using localhost:5000 as the registry frompods inside the microk8s cluster.

Remember that registries deployed on microk8s are insecure and only available to the localhost you are performing development on.  Public registries can be used after changing the Registry environment variable and modifing the deployment.yaml file to point at them.  Again an externally hosted pipeline can be used to trigger downstream testing and release activities.

### Kubernetes minikube

Work In Progress, your milage may vary.

minikube when run using the --driver=none option is typically hosted within an AWS EC2 or similar environment will not deploy its own image registry and will instead leverage the local hosts registry deployed using docker.  This is because the driver none option relies on docker for its container runtime.

The RegistryPort and RegistryIP variables are available for the job template.  After creation of a minikube cluster used the kubectl context command to configure your Kubernetes client side environmnet, ```kubectl config use-context minikube```.  This is then used by the watcher to set the registry variables.

If you intend on using the Ubuntu and minikube stack then the following installation guide might be of help, https://gist.github.com/karlmutch/f6bfc81a08b3815888d632dc572764bc.

### bootstrapping

Having setup the git-watch process the [--stat-persistence-dir] is used to store information about the last commit seen and acted on by the watcher.

The watcher will check the git repositories on a regular basis to poll for new commits and will initiate Kubernetes jobs on the latest observed commit ID.

As each job is run git-watch will generate a namespace based upon the current semantic version set in the README.md file at the root of your repository.  Information about the version tags used can be found in the semver section of this document.  Importantly the semver utility in this package only generates pre-release tags that obey DNS naming rules and this allows Kubernetes to use these identifiers as DNS compliant namespaces.

The example ci\_containerize.yaml example file illustrates a job that will build a docker image containing the source code at the detected commit ID and will push this code to the docker hub repository.  The docker-registry-config secret in this file is used to store the user name and password as described above for completing the docker push operation once the Makisu build is done.

In order to enable console output the following commands are useful:

```
LOGXI='*=INF'
LOGXI_FORMAT='happy,maxcol=1024'
```

Please review https://github.com/mgutz/logxi for more information on logging levels and environment variables.

### usage

<pre><code>
usage:  git-watch [options] [arguments]      Git Commit watcher and trigger (git-watch)       1910aaf935e3a53cf22de4ede292492b77164bdb      2019-07-18_13:01:01-0700

Arguments:

git-watch arguments take the form of a web URL containing the URL for the repository followed
by an optional caret '^' and branch name.  If the caret and branch name are not specified then the
branch name is assumed to be master.

Example of valid arguments include:

  https://github.com/karlmutch/duat.git
  https://github.com/karlmutch/duat.git^master

Options:

  -debug
        Enables features useful for when doing step by step debugging such as delaying cleanup operations etc
  -github-token string
        A github token that can be used to access the repositories that will be watched
  -job-template string
        The Kubernetes job specification stencil template file name that is run on a change being detected, env var GIT_HOME will be set to indicate the repo directory of
the captured repository
  -namespace string
        The namespace that should be used for processing the bootstrap, potentially destructive cleanup might be used on this namespace
  -persistent-state-dir string
        Overrides the default directory used to store state information for the last known commit of the repositories being watched (default "/tmp/git-watcher")
  -v    When enabled will print internal logging for this tool

Environment Variables:

options can also be extracted from environment variables by changing dashes '-' to underscores and using upper case.

log levels are handled by the LOGXI env variables, these are documented at https://github.com/mgutz/logxi
</code></pre>

### downstream CI/CD (keel.sh)

