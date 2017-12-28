#!/bin/sh

set -eu

read -p "input next version: " next_version

gobump set ${next_version} -w cmd/

git commit -am "Checking in changes prior to tagging of version v$next_version"
git tag v${next_version}
git push && git push --tags

goxz -pv=v$(gobump show -r cmd/) -d=./dist/v$(gobump show -r cmd/)
ghr ${next_version} dist/${next_version}