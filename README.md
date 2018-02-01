# bump-md-ver
A version bumping tool for storing iand manipuating the semantic version within a github style markdown file.

This tool also has the ability to generate version string using git branch names.

This tool builds on the Semantic Versioning 2.0 spec, https://semver.org/, to use the build metadata to store the branch and commit hash as part of the generated version string.

Version : <repo-version>0.0.0</repo-version>

# Installation

This go can installed using the following command:

```shell
$ go get -u karlmutch/bump-md-ver
```

# Basic usage

bump-md-ver by default will read your README.md file and will examine it for HTML markup embeeded in the file `&lt;repo-version&gt;[semver]&lt;/repo-version&gt;`.  The text within the tag will be parsed and validated as being valid semver, if this fails the command will exit.  Once parsed the options specified on the bump-md-v command line will be used to morp the version and written back into the file.

The command has the following usage:

<doc-opt>```
Bump MD File Version

Usage:

  bump-md-ver [major | major | minor | dev | extract] [-f=<markdown-file>]

Options:
  -h --help          Show this message.
  -version           Show the version of this software.
  -f=<markdown-file> The github formatted markdown file to be processed [default: README.MD]
```</doc-opt>


