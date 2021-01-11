## Unreal docset generator for Dash

Docset generator for use with [Dash](https://kapeli.com/dash).

Should also work with [Zeal](https://zealdocs.org) and in a regular browser (just open the generated html files inside the docset directory).

### Requirements

* Go 1.13 (https://golang.org/dl/)
* UnrealEngine (http://www.unrealengine.com)

### Generating the docset

* Run `make`
* Run `./bin/unreal-docset [UnrealEngineInstallDir]/Engine/Documentation/Builds`

The documentation will be scrapped from `docs.unrealengine.com`.
Scrapping requires around 7GB bandwidth and around 200k http requests.
Due to rate limitation the scrapping might take a while (up to an hour).

The resulting docset should be around 5GB in size.

### Known issues

* Page [types](https://kapeli.com/docsets#supportedentrytypes) are not always inferred correctly.
