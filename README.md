# Developer utilities and tools (duat)   Alpha

Version : <repo-version>0.4.0</repo-version>

duat is a set of tools useful for automating workflows operating on common software artifacts such as git branches and tags, semantic versioning, and docker image delivery.  duat is a work in progress experiment in using Go to manage the entire software lifecycle removing scripting and other DSLs typically used for building, releasing, and finally deploying software.

This repository also delivers go import packages for handling meta data and artifacts associated with software development activities.

duat make assumptions about naming of docker images, and semantic versioning.  duat provides tools and assistance, but intentionally does not impose and end-to-end automation solution, for CI/CD.

# The name duat

Duat was the name given by the ancient egyptians to the underworld.  As the sun set each day and travelled through the underworld being regenerated it would cast light on to the souls nearby bringing them to life for a period of time as it passed by.  While passing through the duat these tools hope to light each of the hours in your build pipeline.

# Motivating use-cases

A user of duat wishes to develop and build software producing distinct tagged docker images for every build and, clean up built docker images of previous development versions as development progresses.

A user wishes to deliver software packaged using docker images to an AWS ECR image repository.

A user wishes to deploy containerized software into an Istio, or other k8s based service mesh.

Many existing cloud based platforms exist today to address requirements such as the above.  This has led to divided islands of functionality, such as Travis, that require integration with each other so that credentials and other artifacts required by workflow automation is shared between these platform. Costly and fragile integration falls to the developer and users which is time consuming and complex.  duat builds upon the observation that many developers are already operating in a cloud based environment and what is needed is a simple set of tools, if a set of simplifing assumptions is made for addressing the above use cases, especially if you already have containerized builds, and tests.

For example the following workflow might be used to compile duat itself:

```shell
go run ./build.go -r cmd > /tmp/compiled.lst
# test is run and passes
cat /tmp/compiled.lst | go run ./cmd/github-release/github-release.go -
```

# Conventions assumptions

The tools and packages within this project rely on a couple of conventions and assumptions.

1. git

    git is the primary source code management tool
    
    git tags are semver compliant

2. release targets

    can be containers stored using docker registries
    
    can be github releases

3. semantic versioning

    using prefix characters are not semver compliant and are not used

    semver pre-release versions are sortable and are ordered

4. containerization

    builds are performed using containers

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
karlmutch/no-code/noserver:0.1.0-89-bzip2-1eqpVy
```

represents a docker image from the github.com/karlmutch/no-code repository, and the noserver component, that is a pre-release of 0.1.0 and was generated from the 89_bzip2, or 89-bzip2 branch with the pre-release timestamp of 1eqpVy.

The git repository name is obtained by the git package

The version portion of the semver is wrangled by the semver package using the README.md file as the authortative source.

The pre-release portion is obtained from git using the branch name, and the trailing portion '1eqpVy' is a Base 62 encoding, modified to allow sorting is date time order of the string, pre-release stamped within the README.md file.

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
0.0.2-00-new-feature-1eqGTxa
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

1. as standalone utilities to do very discret actions when invoked by bash scripts for the command line.

2. as packages inside go source files, the source files being executed in the same manner as shell scripts.

The go packages this project implements are intended to be employed when using Go as a scripting language, as an alternative to bash scripts.  Information about using Go as a shell scripting language can be found at, https://blog.cloudflare.com/using-go-as-a-scripting-language-in-linux/.

The general idea is to produce both libraries for development artifact handling and also tools that can be invoked to perform administrative functions related to software release and build activities.

# Installation using Github binaries

duat has regular releases of the stable head of the git repo.  The release consists at this time of precompiled Linux x86_64 binaries that can be found at https://github.com/karlmutch/duat/releases.

# Installation using go get

duat is go gettable with the command tools compiled using the go tools.  The following commands will suffice for most Go environments.

```
go get github.com/karlmutch/duat
go install github.com/karlmutch/duat/semver
go install github.com/karlmutch/duat/github-release
go install github.com/karlmutch/duat/image-release
go install github.com/karlmutch/duat/stencil
```

# Building duat from source

## Prerequisites

```
go get github.com/erning/gorun
sudo mv ~/go/bin/gorun /usr/local/bin/
echo ':golang:E::go::/usr/local/bin/gorun:OC' | sudo tee /proc/sys/fs/binfmt_misc/register
```

### Docker based Release builds

Using build.sh

### Development builds

Using build.go

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

## image-exists

A tool for testing if a local docker image exists for the given module, and version.  This tool would typically be run within a source directory for a cmd.  It extracts the module name from the directory name, along with the git information, and version and then tests to determine if a local docker image is present that matches the code version.

Output for this command is exit code 1 for when there is not matching image, and exit code 0 for when an image exists.

<pre><code>
usage:  ../../cmd/image-exists/image-exists.go [options]       image exists test tool (image-exists)       unknown      unknown

Options:

  -module string
          The name of the component that is being used to identify the container image, this will default to the current working directory (default ".")
            -v    When enabled will print internal logging for this tool

            Environment Variables:

            options can also be extracted from environment variables by changing dashes '-' to underscores and using upper case.

            log levels are handled by the LOGXI env variables, these are documented at https://github.com/mgutz/logxi
</code></pre>

## docker-groom

## image-release

<code><pre>
image-release is used to tag a pre-release image version, as a released semantic version, and then optionally push the resulting image to an AWS ECR repository.  The pre-release version will also be cleared, if no pushes failed, from the README.md file resulting in an offical release.  Exit codes are used to denote success or failure of this command to tag then push an image.

age:  image-release [options]       docker image release tool (image-release)       a0412bff1a2de2ed92ef548a79f76a65d64d720e      2018-03-20_21:22:22-0700

Options:

  -f string
        The file to be used as the source of truth for the existing, and future, version (default "README.md")
  -module string
        The name of the component that is being used to identify the container image, this will default to the current working directory (default ".")
  -production
        When enabled will generate tools etc as production releases by removing pre-release version markers
  -release-repo string
        The name of a remote image repository, this will default to no remote repo
  -v    When enabled will print internal logging for this tool

Environment Variables:

options can also be extracted from environment variables by changing dashes '-' to underscores and using upper case.

log levels are handled by the LOGXI env variables, these are documented at https://github.com/mgutz/logxi
</pre></code>

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
```

Templates also support functions from masterminds.github.io/sprig.  Please refer to that github website for more information.
