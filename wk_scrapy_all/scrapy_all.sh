#! /bin/bash

set -eu -i pipefail

curl -o br08601.keg \
"https://www.genome.jp/kegg-bin/download_htext?htext=br08601.keg&format=htext&filedir="

grep "^E" br08601.keg | awk '{print $2}' |
xargs -i -n 100 Pathway Get &> get_pathway.log
