
# Repository README Template

<!-- This is a README template used as a basis for most repositories hosted here. -->
<!-- This repository has two branches: -->
<!-- main       - Contains the README and other default files -->

<!-- Documentation Setup/Pull/Edit/Push -->
<!-- ----------------------------------------- -->

<!-- We include the Github's wiki repository as a subtree of the project's repository. -->
<!-- (Using this [link](https://gist.github.com/SKempin/b7857a6ff6bddb05717cc17a44091202)) -->
<!-- Please check the raw version of this README, contains comments with appropriate -->
<!-- commands for pushing/pulling the documentation subtree. -->

<!-- Add the initial wiki in a subtree (normally ':branch' should be 'main'): -->
<!-- git subtree add --prefix docs/ https://github.com/:user/:repo.wiki.git :branch --squash -->

<!-- Pull latest changes in the wiki -->
<!-- `git subtree pull --prefix docs/ https://github.com/:user/:repo.git master --squash` -->

<!-- Push your changes to the wiki -->
<!-- `git subtree push --prefix docs/ https://github.com/:user/:repo.git :branch` -->

<!-- Badges -->
<!-- ----------------------------------------- -->

<!-- Assuming the majority of them being written in Go, most of the badges below -->
<!-- are dedicated to Go things. However, several words MUST BE REPLACED in the lines below:  -->
<!-- - :user         => The user or organization owning the repository (here, reeflective) -->
<!-- - :repo         => The name of the repository created from this template -->
<!-- - :branch       => Some badges use a specific branch (such codecov tools) -->
<!-- - :path/:file   => Some badges require paths to a specific file (replace all nodes starting with : ) -->
![Github Actions (workflows)](https://github.com/:user/:repo/workflows/:workflow-name/badge.svg?branch=:branch-name)
[![Go module version](https://img.shields.io/github/go-mod/go-version/:user/:repo.svg)](https://github.com/:user/:repo)
[![GoDoc reference](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/:user/go/:repo)
[![GoReportCard](https://goreportcard.com/badge/github.com/:user/:repo)](https://goreportcard.com/report/github.com/:user/:repo)
[![codecov](https://codecov.io/gh/:user/:repo/branch/master/graph/badge.svg)](https://codecov.io/gh/:user/:repo)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

## DELETE ME (TODO)

### Initial remote setup

This template is tied to github.com/reeflective/template, therefore it has this path
set as git origin ('git remote show origin'). When using this template, please do:

```bash
git remote remove origin
git remote add https://github.com/:user/:repo
```

-----

## Summary

-----

## Install

-----

## Documentation

-----

## Status
