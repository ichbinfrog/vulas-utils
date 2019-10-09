
[![Go Report Card](https://goreportcard.com/badge/ichbinfrog/vulas-utils)](https://goreportcard.com/report/github.com/ichbinfrog/vulas-utils) 
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/ichbinfrog/vulas-utils/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/ichbinfrog/vulas-utils?status.svg)](https://godoc.org/github.com/ichbinfrog/vulas-utils)
[![GoVersion](https://img.shields.io/badge/goversion-1.13.1-brightgreen.svg)](https://github.com/gojp/goreportcard/blob/master/LICENSE)

# Vulas-utils

A CLI that is meant to help automatically manage the vulnerability-assessment-tool helm chart by allowing for the following features:
- Upgrading releases with database schema changes
- Configure the admin chart to serve a specific release
- Load up data into the vulnerability database (either by dumps or manual)

## . Installation
You can either download a release or assuming you already have a recent version of Go installed, pull down the code with `go get`:
```sh
go get github.com/ichbinfrog/vulas-utils
```
