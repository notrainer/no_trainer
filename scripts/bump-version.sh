#!/usr/bin/env bash
case $1 in
  "minor")
    version=$(echo $VERSION | cut -d'.' -f1)'.'$(($(echo $VERSION | cut -d'.' -f2)+1))'.0'
    ;;
  "major")
    version=$(($(echo ${VERSION} | cut -d'.' -f1)+1))'.0.0'
    ;;
  "patch"|"build"|"")
    version=$(echo $VERSION | awk -F'.' '{ print $1"."$2 }').$(($(echo $VERSION | cut -d'.' -f3)+1))
    ;;
  *)
    echo "command [$1] does not exist" >/dev/stderr
    exit 1
esac

echo "raise version [${VERSION}] to version [${version}]"

if [[ $OSTYPE == linux* ]]; then
  sed -i 's/VERSION=.*/VERSION='${version}'/' .envrc
elif [[ $OSTYPE == darwin* ]]; then
  sed -i '' 's/VERSION=.*/VERSION='${version}'/' .envrc
fi

if type direnv >/dev/null 2>&1; then
  direnv allow
fi
