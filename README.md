## Unreal docset generator for Dash

Docset generator for use with [Dash](https://kapeli.com/dash).

Should also work with [Zeal](https://zealdocs.org) and in a regular browser (just open the generated html files inside the docset directory).

### Requirements

* Go 1.13 (https://golang.org/dl/)
* UnrealEngine (http://www.unrealengine.com)

### Generating the docset

* Run `make`
* Run `./bin/unreal-docset UnrealEngineInstallDir/Engine/Documentation/Builds`

The API documentation will be build from the archives in `Engine/Documentation/Builds`.
Therefore the path to `Engine/Documentation/Builds` is required, this directory should be located i your UE4 install dir.

The remaining documentation will be scrapped from `docs.unrealengine.com`.
Scrapping requires around 3GB bandwidth and around 100k http requests.
Due to rate limitation the scrapping might take a while (up to an hour).

The resulting docset should be around 4.5GB in size.

### Known issues

* Page [types](https://kapeli.com/docsets#supportedentrytypes) are not always inferred correctly.
