# GitHub Release Redirector

A simple Go web server that redirects a configured list of paths to the latest release asset on a GitHub repository. Useful for maintaining one link that always points to the latest release so that downloading is easy and painless for end users who have trouble navigating GitHub, as well as automated procedures such as CI pipelines.

## Setup

Create a configuration file that lists the files to serve and the GitHub repository to redirect each file to. The program will attempt to read `config.json` by default, but this can be configured using a command-line argument. A sample configuration is available as [`example_config.json`](https://github.com/kdrag0n/github-release-redirector/blob/master/example_config.json).
