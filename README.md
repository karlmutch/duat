# Developer utilities and tools (duat) Beta

Version : <repo-version>0.16.0</repo-version>

duat is a set of tools useful for automating the bootstrapping of containerized workflows.  duat includes tools for working with software artifacts such as git branches and tags, semantic versioning, and docker image delivery.  duat is a work in progress experiment in using Go, and Kubernetes to manage portions of container centric software lifecycles, helping to remove proprietary tooling, scripting, and other DSLs typically used for building, releasing, and deploying software.

This repository also includes go import packages for handling meta data and artifacts associated with software development activities.

duat is opinionated about naming of docker images, and semantic versioning.  duat provides tools and assistance specifically for go based development, but intentionally does not impose an end-to-end CI/CD automation solution.  Downstream workflow choices are determined by the user not duat.

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

git-watch has been retired.  The preferred method to deal with git driven builds and file watch based builds is to make use of tilt.dev.

## license-detector

license-detector is useful for performing data mining of code bases and performing license detection where standard license information is not made available via standard license selection using the github license feature for example.

This tool is useful as an investigative tool for situations where incomplete formal licensing information is available.

The role of this tool within an SPDX based workflow is intended to be a catch-all tool for when formal license infomation is not stored a metadata, for example when the API documented at https://docs.github.com/en/rest/reference/licenses is not able to return useful information.

At a minimum this tool should be run against code bases using the the `short` option and the summary results quickly scanned to discover if any non-permitted licenses were found.  If so then a full report should be run to discover violations.

It is recommended that this tool be used in conjunction with a tool such as, https://github.com/google/go-licenses, which will restrict itself to using the Go vendor directory and looking for explicit license files.  

If you wish to extract the SPDX ids for repositories based upon the github metadata fields then the offical github CLI tool can be used as follows:

```
gh api repos/karlmutch/duat/license | jq ".license.spdx_id"
```

While the github API can display this information Go modules come from a large number of sources and this is where the license-detector comes in handy.

To assist in creating manifests and lists useful to the license scanning tools the reader might find the `go list` tool useful for creating lists of packages, for example:

```shell
go list -mod=mod -m all
```

### Installation

license-detector can installed using the following command:

```shell
$ go get -u karlmutch/duat/cmd/license-detector
```

### Basic usage

This tool scans directories, and individual files for license files and materials useful in identifing licenses using hashing, and fuzzing techniques to produce a list of SPDX identified licenses along with confidence measure for the licenses potentially detected.

This utility wraps the licensing tooling documented at https://github.com/go-enry/go-license-detector, and the license database found at https://github.com/spdx/license-list-data.

This tool will output a list of all directories and their matched licenses when using the non-short form option.  Using the short form option a list of licenses that were detected will be produced and the maximum confidence seen for that license in the scanned directories will be output.  All license identification uses the SPDX codes.

The command has the following usage:

<doc-opt><code>
usage:  /tmp/go-build149289568/b001/exe/license-detector [options]       license scanning and detection tool (license-detector.go)       unknown      unknown

Arguments

  -i string
        The location of files to be scanned for licenses, or a single file (default ".")
  -input string
        The location of files to be scanned for licenses, or a single file (default ".")
  -o string
        The location of the license detection SPDX inventory output file, the referenced directory will be skipped from license checks (default "license-detector.rpt")
  -output string
        The location of the license detection SPDX inventory output file, the referenced directory will be skipped from license checks (default "license-detector.rpt")
  -r    Descend all directories looking for licensed source code (default true)
  -recurse
        Descend all directories looking for licensed source code (default true)
  -s    Output just a summary of the license types detected
  -short
        Output just a summary of the license types detected
  -v    Print internal logging for this tool
  -verbose
        Print internal logging for this tool

Environment Variables:

options can also be extracted from environment variables by changing dashes '-' to underscores and using upper case.

log levels are handled by the LOGXI env variables, these are documented at https://github.com/karlmutch/logxi
</code></doc-opt>



Copyright © 2018-2021 The duat authors. All rights reserved. Issued under the MIT license.
