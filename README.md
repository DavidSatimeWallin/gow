# GOW - a Go Wiki

GOW is a local wiki software built to easily structure local notes, links and general knowledge. It uses a JSON-based file-structure to store the data. The files are encrypted though so the JSON format isnt very useful outside of this software.

If you *double click* on any word or text string you get the choice to create a new articled with that title. Any text that matches an existing articles title will automatically get linked.

In the top there's a search area which uses a ranked fuzzy search. It will rank the content for you and show you a list of hits ranked on importance to you.

## Demo

To see a quick demo visit https://www.useloom.com/share/1a9c0721485e42ecad2606dc2d52aa4f

## Install & Use

*OBS! This is completely untested on windows OBS!*

### Install

#### Binary

In the `binaries` -folder you should find compiled binaries for most platforms. Just download the one for your platform and run it. This will launch the wiki on `localhost:9090`

#### Source

Simply `cd` into your `$GOPATH` and type `go get github.com/dvwallin/gow` then `cd $GOPATH` and `cd github.com/dvwallin/gow` and do `go install`

### Usage

Running the wiki without commands will create a folder in $HOME named gow.bucket which will contain all the data. If you want the bucket located elsewhere simply append -bucket <path-to-bucket>.

## Help

```
  -bucket string
    	the folder in which data should be stored (default "./gow.bucket")
  -host string
    	the host on which to host the web interface (default "0.0.0.0")
  -key string
    	the secret key you want to use for encryption (default "d51b2bf666420e87ab91d08ef07f2e08")
  -port string
    	the port you want to run GOW on (default "9090")
```

## Roadmap

 * [ ] Remote syncing of the bucket
 * [ ] Connect to other peoples wikis
 * [ ] Download an archive
 * [ ] Better markdown support
 
## Contribute

Any help at all is just awesome!

## License

MIT License

Copyright (c) 2016 David Satime Wallin <david@dwall.in>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
