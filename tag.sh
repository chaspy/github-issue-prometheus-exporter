#!/bin/bash

# タグ一覧を取得して、最新のタグを取得
git fetch --tags --prune-tags --prune
latest_tag=$(git tag -l --sort=-v:refname | head -n 1)

# タグがない場合、v1.0.0 として初期化
if [ -z "$latest_tag" ]; then
  latest_tag="v1.0.0"
fi

# 最新のタグからパッチバージョンを 1 つ上げる
new_tag=$(echo "$latest_tag" | awk -F. '{ printf("%s.%s.%s", $1, $2, $3+1) }')

git tag -a ${new_tag} -m "Release ${new_tag}"

git push --tags
