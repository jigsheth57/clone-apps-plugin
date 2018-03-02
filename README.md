# Clone Apps Plugin
This CF CLI Plugin will export apps metadata (including service instances & environment variables info), droplets & src code you have permission to access.

#Usage

For human readable output:

```
➜  clone-apps-plugin git:(master) ✗ cf export-apps
Succefully exported apps metadata to apps.json file.
```

Filter by Org Name

```
➜  clone-apps-plugin git:(master) ✗ cf export-apps -o Central
Succefully exported apps metadata to apps.json file.
```

Download metadata & bits:

```
➜  clone-apps-plugin git:(master) ✗ cf export-apps -d download
Number of app bits to download  4
Wrote file:  server-c2c_11d5ec05-8919-481d-833d-48a37384ea2a.src
Wrote file:  timetracking_04571ea8-7741-4a50-a28d-9c2136c2235c.src
Wrote file:  server-c2c_11d5ec05-8919-481d-833d-48a37384ea2a.droplet
Wrote file:  timetracking_04571ea8-7741-4a50-a28d-9c2136c2235c.droplet
Succefully exported apps metadata to apps.json file and downloaded all bits.
```


##Installation
```
For OSX
$ cf install-plugin https://github.com/jigsheth57/clone-apps-plugin/blob/master/bin/osx/clone-apps-plugin?raw=true -f

For Windows 32bit
$ cf install-plugin https://github.com/jigsheth57/clone-apps-plugin/blob/master/bin/win32/clone-apps-plugin.exe?raw=true -f

For Windows 64bit
$ cf install-plugin https://github.com/jigsheth57/clone-apps-plugin/blob/master/bin/win64/clone-apps-plugin.exe?raw=true -f

For Linux 64bit
$ cf install-plugin https://github.com/jigsheth57/clone-apps-plugin/blob/master/bin/linux64/clone-apps-plugin?raw=true -f

```
#####Install from Source (need to have [Go](http://golang.org/dl/) installed)
  ```
  $ go get github.com/cloudfoundry/cli
  $ go get github.com/jigsheth57/clone-apps-plugin
  $ cd $GOPATH/src/github.com/jigsheth57/clone-apps-plugin
  $ go build
  $ cf install-plugin clone-apps-plugin
  ```
