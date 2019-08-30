#!/bin/sh

if [[ "$1" = "release" ]] ; then
	TAG="$2"
	: ${TAG:?"Usage: build_all.sh [release] [TAG]"}

	git tag | grep $TAG > /dev/null 2>&1
	if [ $? -eq 0 ] ; then
		echo "$TAG exists, remove it or increment"
		exit 1
	else
		MAJOR=`echo $TAG |  awk 'BEGIN {FS = "." } ; { printf $1;}'`
		MINOR=`echo $TAG |  awk 'BEGIN {FS = "." } ; { printf $2;}'`
		BUILD=`echo $TAG |  awk 'BEGIN {FS = "." } ; { printf $3;}'`

		`sed -i .bak -e "s/Major:.*/Major: $MAJOR,/" \
			-e "s/Minor:.*/Minor: $MINOR,/" \
			-e "s/Build:.*/Build: $BUILD,/" cloneapps.go`
	fi
fi

GOOS=linux GOARCH=amd64 go build
LINUX64_SHA1=`cat clone-apps-plugin | openssl sha1 | cut -d " " -f2-`
mkdir -p bin/linux64
mv clone-apps-plugin bin/linux64

GOOS=darwin GOARCH=amd64 go build
OSX_SHA1=`cat clone-apps-plugin | openssl sha1 | cut -d " " -f2-`
mkdir -p bin/osx
mv clone-apps-plugin bin/osx

GOOS=windows GOARCH=amd64 go build
WIN64_SHA1=`cat clone-apps-plugin.exe | openssl sha1 | cut -d " " -f2-`
mkdir -p bin/win64
mv clone-apps-plugin.exe bin/win64

GOOS=windows GOARCH=386 go build
WIN32_SHA1=`cat clone-apps-plugin.exe | openssl sha1 | cut -d " " -f2-`
mkdir -p bin/win32
mv clone-apps-plugin.exe bin/win32

UPDATED_TIME=`date +"%Y-%m-%dT%T%Z"`
cat repo-index.yml.template |
sed "s/osx-sha1/$OSX_SHA1/" |
sed "s/win64-sha1/$WIN64_SHA1/" |
sed "s/win32-sha1/$WIN32_SHA1/" |
sed "s/linux64-sha1/$LINUX64_SHA1/" |
sed "s/_TAG_/$TAG/" |
sed "s/_TIME_/$UPDATED_TIME/" |
cat > repo-index.yml 2>&1
cat repo-index.yml

#Final build gives developer a plugin to install
if [[ "$1" = "release" ]] ; then
	git commit -am "Build version $TAG"
	git tag $TAG
	echo "Tagged release, 'git push --tags' to move it to github, and copy the output above"
	echo "to the cli repo you plan to deploy in"
fi
