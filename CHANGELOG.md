# 0.16.0

IMPROVEMENTS:

* license-detector added to enable fuzzy license detection to assist in detection of unruly licenses
* Many package and module upgrades for Kubernetes 1.22 and other supported packages

BUG FIXES:

* pod file copies would not truncate files correctly as they wrote new contents

DEPRECATED:

* git-watch is retried and now redundant due to tools such as https://tilt.dev and highly recommended in conjunction with Rancher Desktop
