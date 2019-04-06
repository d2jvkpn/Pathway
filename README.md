KEGG pathway process, usage:

#### 1. update local data table  ($EXECUTINGPATH/KEGG_data/KEGG_organism.tsv):

$ Pathway  Update

#### 2. download organisms keg file (s):

$ Pathway  Get  hsa mmu ath

#### 3. get keg file of an organism from local:

$ Pathway  get  hsa

Note: make sure you have download organisms' keg files and achieve to

$EXECUTINGPATH/KEGG_data/Pathway_keg.tar

#### 4. find match species name or code in local data table:

$ Pathway  match  "Rhinopithecus roxellana"

$ Pathway  match  Rhinopithecus+roxellana

$ Pathway  match  rro

#### 5. download pathway html:

$ Pathway  HTML  hsa00001.keg.gz  ./hsa00001

Note: existing html files will not be overwritten

#### 6. convert keg format to tsv  (file or stdout):

$ Pathway  tsv  hsa00001.keg.gz  hsa00001.keg.tsv

output tsv header: gene_id gene_information C_id C_name

KO_id KO_information EC_ids B_id B_name A_id A_name

#### 7. download species keg, convert to tsv and download html files:

$ Pathway  species  Rhinopithecus+roxellana

Note: existing html files will be overwritten
