package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

const HELP = `KEGG pathway process, usage:

1. update local data table  ($EXECUTINGPATH/KEGG_data/KEGG_organism.tsv):
    $ Pathway  Update

2. download organisms keg file (s):
    $ Pathway  Get  hsa mmu ath

3. get keg file of an organism from local:
    $ Pathway  get  hsa
    Note: make sure you have download organisms' keg files and achieve to
    $EXECUTINGPATH/KEGG_data/Pathway_keg.tar

4. find match species name or code in local data table:
    $ Pathway  match  "Rhinopithecus roxellana"
    $ Pathway  match  Rhinopithecus+roxellana
    $ Pathway  match  rro

5. download pathway html:
    $ Pathway  HTML  hsa00001.keg.gz  ./hsa00001
    Note: existing html files will not be overwritten

6. convert keg format to tsv  (file or stdout):
    $ Pathway  tsv  hsa00001.keg.gz  hsa00001.keg.tsv

    output tsv header: gene_id gene_information C_id C_name
	KO_id KO_information EC_ids B_id B_name A_id A_name

7. download species keg, convert to tsv and download html files:
    $ Pathway  species  Rhinopithecus+roxellana
    Note: existing html files will be overwritten

author: d2jvkpn
version: 0.9.1
release: 2019-06-19
project: https://github.com/d2jvkpn/Pathway
lisense: GPLv3  (https://www.gnu.org/licenses/gpl-3.0.en.html)
`

func main() {
	nargs := len(os.Args) - 1

	if nargs == 0 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		fmt.Println(HELP)
		os.Exit(2)
	}

	cmd := os.Args[1]
	ep, _ := exec.LookPath(os.Args[0])
	datatsv := filepath.Dir(ep) + "/KEGG_data/KEGG_organism.tsv"

	switch {
	case cmd == "Update" && nargs == 1:
		Update(datatsv)

	case cmd == "Get" && nargs > 1:
		Get(os.Args[2:])

	case cmd == "get" && nargs == 2:
		ok := Get_local(os.Args[2]+"00001.keg.gz",
			filepath.Dir(ep)+"/KEGG_data/Pathway_keg.tar")

		if !ok {
			os.Exit(1)
		}

	case cmd == "HTML" && nargs == 3:
		DownloadHTML(os.Args[2], os.Args[3], false)

	case cmd == "match" && nargs == 2:
		record, found := Match(strings.ToLower(os.Args[2]), datatsv)

		if found {
			fmt.Printf("Entry: %s\nCode: %s\nSpecies: %s\nLineage: %s\n",
				record[0], record[1], record[2], record[3])
		} else {
			fmt.Fprintln(os.Stderr, "NotFound")
		}

	case cmd == "tsv" && (nargs == 3 || nargs == 2):
		if nargs == 3 {
			ToTSV(os.Args[2], os.Args[3])
		} else {
			ToTSV(os.Args[2], "")
		}

	case cmd == "species" && nargs == 2:
		record, found := Match(formatSpeciesName(os.Args[2]), datatsv)

		if found {
			fmt.Printf("Entry: %s\nCode: %s\nSpecies: %s\nLineage: %s\n",
				record[0], record[1], record[2], record[3])

			log.Printf("querying %s\n", record[1]+"00001.keg")

			ok := getkeg(record[1] + "00001.keg")
			if !ok {
				os.Exit(1)
			}

			ToTSV(record[1]+"00001.keg.gz", record[1]+"00001.keg.tsv")

			DownloadHTML(record[1]+"00001.keg.gz", record[1]+"00001", true)
		} else {
			fmt.Fprintln(os.Stderr, "NotFound")
		}

	default:
		log.Fatal(HELP)
	}
}

func formatSpeciesName(name string) string {
	wds := strings.Fields(strings.Replace(name, "+", " ", -1))
	re := regexp.MustCompile("[A-Za-z][a-z]+")

	for i, _ := range wds {
		if re.Match([]byte(wds[i])) {
			wds[i] = strings.ToLower(wds[i])
		}
	}

	wds[0] = strings.Title(wds[0])
	return (strings.Join(wds, " "))
}

func DownloadHTML(keg, outdir string, overwrite bool) {
	var bts []byte
	var err error

	frd, err := os.Open(keg)
	if err != nil {
		log.Fatal(err)
	}
	defer frd.Close()

	err = os.MkdirAll(outdir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("save html files to %s\n", outdir)

	if strings.HasSuffix(keg, ".gz") {
		var gz *gzip.Reader
		gz, err = gzip.NewReader(frd)
		if err != nil {
			return
		}

		bts, err = ioutil.ReadAll(gz)
	} else {
		bts, err = ioutil.ReadAll(frd)
	}

	re := regexp.MustCompile("PATH:[a-z]+[0-9]+")
	PATHs := re.FindAllString(string(bts), -1)

	for i, _ := range PATHs {
		PATHs[i] = strings.TrimPrefix(PATHs[i], "PATH:")
	}

	var wg sync.WaitGroup
	ch := make(chan struct{}, 10)

	for _, p := range PATHs {
		ch <- struct{}{}
		wg.Add(1)
		go gethtml(p, outdir, overwrite, ch, &wg)
	}
	wg.Wait()
}

func gethtml(p, outdir string, overwrite bool, ch <-chan struct{},
	wg *sync.WaitGroup) {

	defer func() { <-ch; defer wg.Done() }()

	html := outdir + "/" + p + ".html"
	png := outdir + "/" + p + ".png"
	url := "http://www.genome.jp/kegg"
	code := p[:(len(p) - 5)]

	if _, err := os.Stat(html); err == nil && !overwrite {
		return
	}

	htmlurl := fmt.Sprintf(url+"-bin/show_pathway?%s", p)
	pngurl := fmt.Sprintf(url+"/pathway/%s/%s.png", code, p)

	htmlresp, err := http.Get(htmlurl)
	if err != nil {
		log.Println(err)
		return
	}
	defer htmlresp.Body.Close()

	htmlbody, err := ioutil.ReadAll(htmlresp.Body)
	if err != nil {
		log.Println(err)
		return
	}

	text := string(htmlbody)
	if !strings.HasSuffix(text, "</html>\n") {
		return
	}

	re, _ := regexp.Compile("\\[[\\S\\s]+?\\]")
	text = re.ReplaceAllString(text, "")

	re, _ = regexp.Compile("\\<script[\\S\\s]+?\\</script\\>")
	text = re.ReplaceAllString(text, "")

	re, _ = regexp.Compile("\\<style[\\S\\s]+?\\</style\\>")
	text = re.ReplaceAllString(text, "")

	re, _ = regexp.Compile("\\<link[\\S\\s]+?\\/\\>[ \n]+")
	text = re.ReplaceAllString(text, "\n")

	re, _ = regexp.Compile("\\<table[\\S\\s]+?\\</table\\>")
	text = re.ReplaceAllString(text, "")

	re, _ = regexp.Compile("\\<div[\\S\\s]+?\\</form>")
	text = re.ReplaceAllString(text, "</div>")

	re, _ = regexp.Compile("\\<body\\>[ \n]+")
	text = re.ReplaceAllString(text, "<body>\n<div align=\"center\">\n")

	text = strings.Replace(text, fmt.Sprintf("/kegg/pathway/%s/", code), "", 1)

	text = strings.Replace(text, "/dbget-bin/www_bget?",
		"https://www.genome.jp/dbget-bin/www_bget?", -1)

	pngresp, err := http.Get(pngurl)
	if err != nil {
		log.Println(err)
		return
	}
	defer pngresp.Body.Close()

	pngbody, err := ioutil.ReadAll(pngresp.Body)
	if err != nil {
		log.Println(err)
		return
	}

	err = ioutil.WriteFile(png, pngbody, 0664)
	if err != nil {
		log.Println(err)
		return
	}

	err = ioutil.WriteFile(html, []byte(text), 0664)
	if err != nil {
		log.Println(err)
		return
	}
}

func ToTSV(keg, tsv string) {
	input, err := NewCmdInput(keg)
	if err != nil {
		log.Fatal(err)
	}

	defer input.Close()

	var TSV io.Writer

	if tsv != "" {
		err = os.MkdirAll(filepath.Dir(tsv), 0755)
		if err != nil {
			log.Fatal(err)
		}

		fwt, err := os.Create(tsv)
		if err != nil {
			log.Fatal(err)
		}

		defer fwt.Close()
		TSV = fwt
	} else {
		TSV = os.Stdout
	}

	var line string
	var fds [11]string

	TSV.Write([]byte("gene_id\tgene_information\tC_id\tC_name" +
		"\tKO_id\tKO_information\tEC_ids\tB_id\tB_name\tA_id\tA_name\n"))

	A := make([]string, 0, 2)
	B := make([]string, 0, 2)

	for input.Scanner.Scan() {
		line = input.Scanner.Text()
		if len(line) < 2 {
			continue
		}

		switch line[0] {
		case 'D':
			tmp := strings.SplitN(strings.TrimPrefix(line, "D      "), "\t", 2)
			if len(tmp) != 2 {
				continue
			}

			copy(fds[0:2], strings.SplitN(tmp[0], " ", 2))

			KOEC := strings.SplitN(tmp[1], " ", 2)
			fds[4], fds[5] = KOEC[0], KOEC[1]

			if strings.Contains(fds[5], " [EC:") {
				x := strings.SplitN(fds[5], " [EC:", 2)
				fds[5] = x[0]
				fds[6] = strings.Replace(x[1], "]", "", 1)
			}

			TSV.Write([]byte(strings.Join(fds[0:], "\t") + "\n"))

		case 'A':
			A = strings.SplitN(line, " ", 2)

		case 'B':
			B = strings.SplitN(strings.Replace(line, "B  ", "B", 1), " ", 2)

		case 'C':
			copy(fds[9:11], A)
			copy(fds[7:9], B)

			tmp := strings.SplitN(strings.TrimPrefix(line, "C    "), " ", 2)
			fds[2], fds[3] = "C"+tmp[0], tmp[1]

			P := make([]string, 2)
			if strings.Contains(fds[3], " [") {
				P = strings.SplitN(fds[3], " [", 2)
				fds[3] = P[0]
				P[1] = strings.TrimSuffix(P[1], "]")
			}

			if P[1] != "" {
				fds[2] = P[1]
			}

		default:
			continue

		}
	}

	if tsv != "" {
		log.Printf("saved %s to %s\n", keg, tsv)
	}
}

func Match(name, datatsv string) (record []string, ok bool) {
	file, err := os.Open(datatsv)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	species := strings.ToLower(formatSpeciesName(name))
	scanner := bufio.NewScanner(file)
	scanner.Scan() // skip header

	for scanner.Scan() {
		record = strings.Split(scanner.Text(), "\t")

		ok = (name == record[1] ||
			species == strings.ToLower(strings.Split(record[2], " (")[0]))

		if ok {
			return
		}
	}

	return
}

func Update(saveto string) {
	log.Printf("Update %s\n", saveto)

	resp, err := http.Get("http://rest.kegg.jp/list/organism")
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}

	err = os.MkdirAll(filepath.Dir(saveto), 0755)
	if err != nil {
		log.Fatal(err)
	}

	fwt, err := os.Create(saveto)
	if err != nil {
		log.Println(err)
		return
	}
	defer fwt.Close()

	fwt.Write([]byte("Entry\tCode\tSpecies\tLineage\n"))
	fwt.Write(body)
}

func Get_local(keg, path string) (ok bool) {
	Cmd := exec.Command("tar", "-xf", path, keg)
	err := Cmd.Run()

	if err != nil {
		log.Printf("failed to get %s from %s\n", keg, path)
	} else {
		ok = true
	}

	return ok
}

func Get(codes []string) {
	ch := make(chan struct{}, 10)
	var wg sync.WaitGroup

	log.Printf("request organism code(s): %s\n", strings.Join(codes, " "))

	for _, v := range codes {
		ch <- struct{}{}
		wg.Add(1)
		go func(p string, ch <-chan struct{}, wg *sync.WaitGroup) {
			defer func() { <-ch; wg.Done() }()
			getkeg(p)
		}(v+"00001.keg", ch, &wg)
	}

	wg.Wait()
}

func getkeg(p string) (ok bool) {
	resp, err := http.Get(fmt.Sprintf("http://www.kegg.jp/kegg-bin"+
		"/download_htext?htext=%s&format=htext&filedir=", p))

	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}

	lines := strings.Split(string(body), "\n")

	if !strings.HasPrefix(lines[len(lines)-2], "#Last updated:") {
		log.Printf("failed to get %s\n", p)
		return
	}

	file, err := os.Create(p + ".gz")
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	gw.Write(body)
	gw.Close()
	log.Printf("saved %s.gz\n", p)

	ok = true
	return
}

type CmdInput struct {
	Name    string
	File    *os.File
	Reader  *gzip.Reader
	Scanner *bufio.Scanner
}

func (ci *CmdInput) Close() {
	if ci.Reader != nil {
		ci.Reader.Close()
	}

	if ci.File != nil {
		ci.File.Close()
	}
}

func NewCmdInput(name string) (ci *CmdInput, err error) {
	ci = new(CmdInput)
	ci.Name = name

	if ci.Name == "-" {
		ci.Scanner = bufio.NewScanner(os.Stdin)
		return
	}

	if ci.File, err = os.Open(ci.Name); err != nil {
		return
	}

	if strings.HasSuffix(ci.Name, ".gz") {
		if ci.Reader, err = gzip.NewReader(ci.File); err != nil {
			return
		}
		ci.Scanner = bufio.NewScanner(ci.Reader)
	} else {
		ci.Scanner = bufio.NewScanner(ci.File)
	}

	return
}
