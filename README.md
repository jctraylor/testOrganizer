# Prerequisites:
1. [Install](https://pkg.go.dev/github.com/andrewkroh/gvm#readme-installation) gvm/go
1. [Install](https://github.com/cli/cli#installation) gh cli (this program runs a gh cli search)
1. [Create](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens#personal-access-tokens-classic) a personal access token that will be used to authenticate the gh cli command. This token will need the following permissions: Repo (all), org:Read, & Gists. Set this token as the value of an env var named 'GH_TOKEN'.
1. Started using golint to lint this thing. Installed with ```go install golang.org/x/lint/golint```. Run with ```golint ./...```

# Run the program
1. From the cloned repo directory run `go run testOrganizer.go` in a terminal
1. You should see an organizedTests-YYYY-MM-DD HH:MM:SS.csv file written to this directory.

# Debug the program: 

In VS Code create a launch.json (run config) with the following information. The env vars are important for being able to run the gh cli command while in the context of the debugger - otherwise it'll fail and say it can't find gh command.

```
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug testOrganizer.go",
      "program": "/path/to/testOrganizer/testOrganizer.go",
      "env": {
        "GH_PATH": "/opt/homebrew/bin/gh",
        "GH_TOKEN": "THIS_VALUE_IS_SECRET"
      }
    }
  ]
}
```

Values to use above:
1. program: run ```pwd``` from root project directory and use the value from the output. 
1. GH_PATH: run ```which gh``` from terminal and use the value from the output. Example value was installed via homebrew
1. GH_TOKEN: This is the token generated in step 3 of the prerequisites. 

With that setup you should be able to go to the Run and Debug tab and run using that config. You can set breakpoints directly in testOrganizer.go file in VS Code. 