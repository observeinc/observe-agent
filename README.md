# Observe Agent

Code for the Observe agent and CLI. The agent code is based on the OpenTelemetry Collector. 

# Build

To run the code you need to have `golang v1.21.7` installed. Then you can run the following command to compile the binary.

```
go build -o observe-agent
```


## Running

To start the observe agent after building the binary run the following command. 

```
./observe-agent start
```


# Releasing

## Goreleaser

First, install goreleaser pro. On MacOS the command is 
```
brew install goreleaser/tap/goreleaser-pro
```

Then, set the following secrets:

- `GORELEASER_KEY`
- `GITHUB_TOKEN`
- `FURY_TOKEN`

Then, create a tag with 
```
git tag v0.1.x
```

Finally, run the `goreleaser` command
```
goreleaser release --clean
```