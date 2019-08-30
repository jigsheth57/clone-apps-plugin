# Clone Apps Plugin
This CF CLI Plugin will export and import apps metadata (including service instances & environment variables info), droplets & src code you have permission to access.

This plugin will create new org and space based on the export metadata file. It will assume that same shared domain is available in new foundation! It also assumes that all managed service based on the export metadata are available and installed. This plugin will create all required service instance (both managed and user provided) and bind to app.

#Usage

For human readable output:

Export metadata from all orgs except **system & p-spring-cloud-services**
```
➜  clone-apps-plugin git:(master) ✗ cf export-apps > export-logs.log 2>&1
```

Export metadata from specific Org Name
```
➜  clone-apps-plugin git:(master) ✗ cf export-apps -o Central > export-logs.log 2>&1
```

Export metadata & download source package & droplet from all orgs except **system & p-spring-cloud-services**:
```
➜  clone-apps-plugin git:(master) ✗ cf export-apps -d download > export-logs.log 2>&1
```

Export metadata & download source package & droplet from specific Org Name:
```
➜  clone-apps-plugin git:(master) ✗ cf export-apps -o Central -d download > export-logs.log 2>&1
```

Import metadata & source package & droplet based on apps.json file in current directory:
```
➜  clone-apps-plugin git:(master) ✗ cf import-apps > import-logs.log 2>&1
```

Import metadata & source package & droplet from specific Org Name based on apps.json file in current directory:
```
➜  clone-apps-plugin git:(master) ✗ cf import-apps -o Central > import-logs.log 2>&1
```

Import metadata & source package & droplet from specific Org Name based on apps.json file in current directory and create additional route based on supplied shared domain:
```
➜  clone-apps-plugin git:(master) ✗ cf import-apps -o Central -ad apps.internal > import-logs.log 2>&1
```

Import metadata & source package & droplet from specific Org Name based on apps.json file in current directory and create additional route based on supplied shared domain and restore original application state:
```
➜  clone-apps-plugin git:(master) ✗ cf import-apps -o Central -ad apps.internal -s true > import-logs.log 2>&1
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
  $ go get github.com/dustin/go-humanize
  $ go get code.cloudfoundry.org/cli/plugin/models
  $ go get github.com/jigsheth57/clone-apps-plugin
  $ cd $GOPATH/src/github.com/jigsheth57/clone-apps-plugin
  $ go build
  $ cf install-plugin clone-apps-plugin
  ```
