#! /bin/bash
# 2019-04-06

set -eu -i pipefail

curl -o br08601.keg \
"https://www.genome.jp/kegg-bin/download_htext?htext=br08601.keg&format=htext&filedir="

grep "^E" br08601.keg | awk '$2!="" {print $2}' |
xargs -i -n 100 Pathway Get &> scrapy_all_pathway_$(date +"%Y%m%d").log
