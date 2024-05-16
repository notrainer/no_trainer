#!/usr/bin/env sh
git tag -f "v${VERSION}"
git push -f origin "v${VERSION}"
