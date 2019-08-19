package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
)

var hpath = "https://www.kegg.jp/kegg-bin/show_pathway?map=%s&show_description=show"
var ipath = "https://www.kegg.jp/kegg/pathway/map/%s.png"

// https://www.kegg.jp/kegg-bin/show_pathway?map=map00520

const HELP = `Download KEGG map html and png, usgae:
   $ KEGG_map_download  <map.list>  [outdir]

note: parallel number is 10, default outdir is ./

author: d2jvkpn
version: 0.2
release: 2019-08-18
project: https://github.com/d2jvkpn/Pathway
lisense: GPLv3  (https://www.gnu.org/licenses/gpl-3.0.en.html)
`

func main() {
	var f, id, outdir string
	var bts []byte
	var err error
	var maps []string
	var ch chan struct{}
	var wg sync.WaitGroup

	if len(os.Args) < 2 {
		log.Println(HELP)
		os.Exit(2)
	}

	f = os.Args[1]
	if len(os.Args) > 2 {
		outdir = os.Args[2]
	} else {
		outdir = "./"
	}

	if bts, err = ioutil.ReadFile(f); err != nil {
		log.Fatal(err)
	}

	if err = os.MkdirAll(outdir, 0755); err != nil {
		log.Fatal(err)
	}

	maps = strings.Fields(string(bts))
	ch = make(chan struct{}, 10)

	for _, id = range maps {
		ch <- struct{}{}
		wg.Add(1)
		go func(id, outdir string, ch <-chan struct{}, wg *sync.WaitGroup) {
			defer func() { <-ch; wg.Done() }()
			gethtml(id, outdir)
		}(id, outdir, ch, &wg)
	}

	wg.Wait()
}

func gethtml(id, dir string) {
	var hres, ires *http.Response
	var hbt, ibt []byte
	var err error

	// html
	if hres, err = http.Get(fmt.Sprintf(hpath, id)); err != nil {
		log.Printf("failed to get html: %v: %s\n", err, id)
		return
	}
	defer hres.Body.Close()

	if hbt, err = ioutil.ReadAll(hres.Body); err != nil {
		log.Printf("failed to read html: %v: %s\n", err, id)
		return
	}

	if strings.Contains(string(hbt), "does not exist") {
		log.Printf("map not exist: %s\n", id)
		return
	}

	//
	if ires, err = http.Get(fmt.Sprintf(ipath, id)); err != nil {
		log.Printf("failed to get image: %v: %s\n", err, id)
		return
	}
	defer ires.Body.Close()

	if ibt, err = ioutil.ReadAll(ires.Body); err != nil {
		log.Printf("failed to read image: %v: %s\n", err, id)
		return
	}

	ioutil.WriteFile(path.Join(dir, id+".html"), hbt, 0644)
	ioutil.WriteFile(path.Join(dir, id+".png"), ibt, 0644)
}
