# Developer utilities and tools (duat)

Version : <repo-version>0.1.0</repo-version>

duat is intended for use by developers implementing workflows operating on common software artifacts such as git branches and tags, semantic versioning, and container delivery.

This repository delivers tools and go import packages for handling meta data and artifacts produced by software development activities.

The motivation for this project is to supply reusable software that can be employed when automating build and delivery activities using Go based software.

Using Go to automate CI/CD style activities is an attempt to address the following:

1. hosted CI/CD solutions create a needless separation with new styles of cloud based development and containers

2. using bash and CI/CD DSLs to automate is disruptive and costly to do

The invdividual tools this projects provides can be run as standalone utilities to do very discret actions when invoked by bash scripts for the command line.

The go packages this project implements are intended to be employed when using Go as a scripting language, as an alternative to bash scripts.  Information about using Go as a shell scripting language can be found at, https://blog.cloudflare.com/using-go-as-a-scripting-language-in-linux/.

The general idea is to produce both libraries for development artifact handling and also tools that can be invoked to perform administrative functions related to software release and build activities.

# The name duat

Duat is the name given by the egyptians to the mythic underworld.  As the sun set each day and travelled through the underworld being regenerated it would cast light on to the souls nearby bringing them to life for a period of time as it passed by.  While passing through the duat I hope these tools shine some light on your travels.

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

  semver [major | major | minor | pre | extract | apply] [-f=<input-file>] [-t=[<target-file>,...]]

Options:
  -h --help              Show this message.
  -version               Show the version of this software.
  -f=<input-file>        A file containing an HTML repo-version tag to be morped or queried [default: README.md]
  -t=<target-file>,...   A comma seperated list of files that will be examined for version tags and modified based upon the input-file version
</code></doc-opt>
