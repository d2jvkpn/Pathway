import os, bs4
from bs4 import BeautifulSoup
from html5print import HTMLBeautifier

HELP = '''Preprocess KEGG map html, usage:
    $ python3 KEGG_map_process.py  <input.html>  <out.html>

author: d2jvkpn
version: 0.2
release: 2019-08-18
project: https://github.com/d2jvkpn/Pathway
lisense: GPLv3  (https://www.gnu.org/licenses/gpl-3.0.en.html)
'''

if len(os.sys.argv) != 2:
    print(HELP)    
    os.sys.exit(2)

hp, out = os.sys.argv[1:3]

with open(hp, "r") as f: soup = BeautifulSoup(f.read(), 'html5lib')

for el in soup.find_all("table"): el.decompose()

achors = ["Pathway menu", "Organism menu", "Pathway entry", "Hide description",
"User data mapping"]

for el in soup.find_all("a"):
    if el.get_text() in achors: el.decompose()

for el in soup.find_all("script"):
     el.decompose()

el = soup.find("style")
if not(el is None): el.decompose()
el = soup.find("link")
if not(el is None): el.decompose()
el = soup.find("div", attrs={"id":"poplay"})
if not(el is None): el.decompose()

for el in soup.find_all("area"):
    if el.attrs["shape"] == "rect":
        if "show_pathway" in el.attrs["href"]:
            el.attrs["class"] = "pathway"
        else:
            el.attrs["class"] = "enzyme"
    elif el.attrs["shape"] == "poly":
        el.attrs["class"] = "reaction"
    elif el.attrs["shape"] == "circle":
        el.attrs["class"] = "compound"

    if "onmouseout" in el.attrs: del el.attrs["onmouseout"]
    if "onmouseover" in el.attrs: del el.attrs["onmouseover"]

    if "href" in el.attrs:
        el.attrs["href"] = "https://www.kegg.jp" + el.attrs["href"]

img = soup.find("img")
img.attrs["src"] = os.path.basename(img.attrs["src"])

els = list(soup.find("body").children)
soup.find("body").clear()
for el in els:
    if not isinstance(el, bs4.element.NavigableString):
       soup.find("body").append(el)

soup.find("body").attrs["style"] = "margin: 0 10%"

with open(out, "w") as f:
    f.writelines(HTMLBeautifier.beautify(str(soup), indent=2))

# html newline: "&#13;"
# html quote: "&quot;"
# length(a)==4{print "region", $2 , "46x18+"a[1]"+"a[2] }
# length(a)==8{print "draw", $2 , "line;"a[1]","a[2]";"a[3]","a[4] }'
