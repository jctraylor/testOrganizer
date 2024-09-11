Prerequisites:
1. [Install](https://pkg.go.dev/github.com/andrewkroh/gvm#readme-installation) gvm/go
1. [Install](https://github.com/cli/cli#installation) gh cli (this program runs a gh cli search)
1. [Create] a personal access token that will be used to authenticate the gh cli command. This token will need the following permissions: Repo (all), org:Read, & Gists. Set this token as the value of an env var named 'GH_TOKEN'.

Run the program
1. From the cloned repo directory run `go run testOrganizer.go` in a terminal
1. You should see an organizedTests.csv file written to this directory.
