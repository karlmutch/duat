# Developer utilities and tools (duat)

Version : <repo-version>0.1.1-03-scripting-example-1eu4IY</repo-version>

duat is intended for use by developers implementing workflows operating on common software artifacts such as git branches and tags, semantic versioning, and docker image delivery.

This repository delivers tools, and go import packages for handling meta data and artifacts produced by software development activities.

duat is a work in progress experiment in using Go across the entire software lifecycle removing scripting and other DSLs typically used for building and releasing software.

duat is highly opinionated about naming of artifacts and semantic versioning.

# Motivating use-case

A developer wishes to develop and build software producing distinct tagged docker images for every build and, cleaning up built docker images of previous development versions as development progresses.

# Introduction

The motivation for this project is to supply software that can be employed when automating build and delivery activities using Go based projects.

Using Go to automate CI/CD style activities is an attempt to address the following:

1. hosted CI/CD solutions create a needless separation with new styles of cloud based development and containerization

2. using bash and vendor specific CI/CD DSLs to automate is disruptive and hard to debug

The individual tools this projects provides can be run in multiple ways:

1. as standalone utilities to do very discret actions when invoked by bash scripts for the command line.

2. as packages inside go source files, the source files being executed in the same manner as shell scripts.

The go packages this project implements are intended to be employed when using Go as a scripting language, as an alternative to bash scripts.  Information about using Go as a shell scripting language can be found at, https://blog.cloudflare.com/using-go-as-a-scripting-language-in-linux/.

The general idea is to produce both libraries for development artifact handling and also tools that can be invoked to perform administrative functions related to software release and build activities.

# Prerequisites

## Installation

```
go get github.com/erning/gorun
sudo mv ~/go/bin/gorun /usr/local/bin/
echo ':golang:E::go::/usr/local/bin/gorun:OC' | sudo tee /proc/sys/fs/binfmt_misc/register
```

# The name duat

Duat is the name given by the egyptians to the mythic underworld.  As the sun set each day and travelled through the underworld being regenerated it would cast light on to the souls nearby bringing them to life for a period of time as it passed by.  While passing through the duat I hope these tools shine some light on your travels.

# Conventions

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

    builds are performed using containerized workflows

# Versioning

duat generates and runs builds using containers with a repository name that represents the github repository name, and tagged using the branch name with no version.

By default the version number of the current repo is stored within the README.md file.  Any other file can be used, for example VERSION as the developers choice.  A typical workflow for versioning is to start by using semver to increment the version based upon the major, minor, patch changes and then apply the new version of the existing README.md file.  If you are doing development then the first step is to use github to generate or identify a ticket and then to create a branch using the tick identifier as the branch name with a description, having done this a git checkout would be used to obtain the branch and then the semver tools use to increment the version string, typically 'semver patch' and follow it up with setting the pre-release version 'semver pre' to add the pre-release tag.  As changes are made and new compiles are successful the 'semver pre' command can continue to be used between succesful compiles if needed to generate new versions within docker for example.  This is useful when doing testing within an existing kubernetes cluster where upgrades of the software are done to test services.

duat generates and software under management using the semantic version, the branch name, and if present the pre-release identifier.  For example:

```
karlmutch/no-code/noserver:0.1.0-89-bzip2-1eqpVy
```

represents a docker image from the github.com/karlmutch/no-code repository, and the noserver component, that is a pre-release of 0.1.0 and was generated from the 89_bzip2, or 89-bzip2 branch with the pre-release identifier of 1eqpVy.

The git repository name is obtained by the git package

The version portion of the semver is wrangled by the semver package using the README.md file as the authortative source.

the pre-release portion is obtained from git using the branch name, and the trailing portion '1eqpVy' is a Base 62 encoding, modified to allow sorting is date time order of the string, pre-release stamped within the README.md file.

## Utilities and Tools

### semver
A version handling tool for storing and manipuating a semantic version (semver) within files.  The semantic version is expressed using an HTML formatted repo-version tag.

This tool also has the ability to generate semver compliant version strings using git branch names when the pre command is used.

This tool applies the Semantic Versioning 2.0 spec to version strings in files, https://semver.org/.  When the pre command is used the version bumping will append to the version string an increasing pre-release string that can be used to sort the versions creating precedence for versions when they are used with containers or other assets.

# Installation

semver can installed using the following command:

```shell
$ go get -u karlmutch/duat/cmd/semver
```

# Basic usage

semver by default will read your README.md file and will examine it for HTML markup embeeded in the file `&lt;repo-version&gt;[semver]&lt;/repo-version&gt;`.  The text within the tag will be parsed and validated as being valid semver, if this fails the command will exit.  Once parsed the options specified on the semver command line will be used to morph the version and written back into the file.

semver can also be used with the apply option to modify files based upon the version within an authorative file.  When this option is used not changes are made to the existing input file.  This command is only for propagating an existing version to other files.

semver will output to stdout the new version number, except for the apply command where you will get the current version applies to the target-file list.

The command has the following usage:

<doc-opt><code>
Semantic Version tool

Usage:

  semver [major | major | minor | pre | extract | apply] [-f=<input-file>] [-t=[&lt;target-file&gt;,...]]

Options:
  -h --help              Show this message.
  -version               Show the version of this software.
  -f=&lt;input-file&gt;        A file containing an HTML repo-version tag to be morped or queried [default: README.md]
  -t=&lt;target-file&gt;,...   A comma seperated list of files that will be examined for version tags and modified based upon the input-file version
</code></doc-opt>
