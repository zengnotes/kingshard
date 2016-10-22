#!/bin/sh
ProjectPath=`pwd`
export GOPATH=$ProjectPath/vendor:$ProjectPath:$GOPATH

if [ -z "$1" ]
then
	echo "place select a version to build...\n\n"
	exit 1
fi

if [ "X$1" = "Xtest" ]
then
	go test -v $2
	exit 0
fi

if [ "X$1" = "Xbench" ]
then
	go test -test.bench=".*"
	exit 0
fi

rm -rf $ProjectPath/bin/*

cd $ProjectPath
echo  $1 "version building..."
##go get github.com/garyburd/redigo/redis
##go get github.com/zengnotes/utility
##go get gopkg.in/yaml.v2
##-gcflags "-N -l"
go build -ldflags "-X config.VERSION=$1 "  -o ./bin/main ./src/cmd/kingshard
##go build -ldflags "-X main._VERSION_ '$1' "  -o ./bin/collection ./src/bin/collection

if [ "X$1" = "X1" ]
then
cd $ProjectPath
./bin/main
else
cd $ProjectPath/bin
mkdir etc
cp -r ../src/etc/ks* ./etc/
mkdir log
##cp -rf ../src/public public
./main
fi

#调试的时候使用
echo  $1 "version build finish"