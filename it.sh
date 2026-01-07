set -eu

iq="go run main.go"
ot=/tmp/iq-it

mkdir -p $ot

##
##
it=IT001
echo "==> $it: source stdin"
cat examples/01_chain/content/doc1.txt | $iq agent -f ./examples/01_chain/run.yml -o $ot/$it.txt

##
##
it=IT002
echo "==> $it: source mount dir"
$iq agent -f ./examples/01_chain/run.yml -I examples/01_chain/content -o $ot/$it.txt

##
##
it=IT003
echo "==> $it: source mount dir (files)"
$iq agent -f ./examples/01_chain/run.yml -I examples/01_chain/content -o $ot/$it.txt doc1.txt doc2.txt

##
##
it=IT004
echo "==> $it: source file"
$iq agent -f ./examples/01_chain/run.yml -o $ot/$it.txt examples/01_chain/content/doc1.txt

##
##
it=IT004
echo "==> $it: source files"
$iq agent -f ./examples/01_chain/run.yml -o $ot/$it.txt examples/01_chain/content/*.txt

##
##
it=IT005
echo "==> $it: feature: chain"
$iq agent -f ./examples/01_chain/run.yml -o $ot/$it.txt examples/01_chain/content/doc1.txt

##
##
it=IT006
echo "==> $it: feature: json schema"
$iq agent -f ./examples/02_json_schema/run.yml -o $ot/$it.txt examples/02_json_schema/content/doc.txt

##
##
it=IT007
echo "==> $it: feature: global state"
$iq agent -f ./examples/03_state/run.yml -o $ot/$it.txt examples/03_state/content/doc.txt

##
##
it=IT008
echo "==> $it: feature: routing"
$iq agent -f ./examples/04_routing/run.yml -o $ot/$it.txt examples/04_routing/content/*.txt

# go run main.go agent -f ./examples/04_routing/run.yml examples/04_routing/content/request2.txt
# go run main.go agent -f ./examples/04_routing/run.yml examples/04_routing/content/request3.txt

##
##
it=IT009
echo "==> $it: feature: routing expression"
$iq agent -f ./examples/05_expression/run.yml -o $ot/$it.txt examples/05_expression/content/*.txt

# go run main.go agent -f ./examples/05_expression/run.yml examples/05_expression/content/doc1.txt
# go run main.go agent -f ./examples/05_expression/run.yml examples/05_expression/content/doc2.txt

##
##
it=IT010
echo "==> $it: feature: foreach"
$iq agent -f ./examples/06_foreach/run.yml -o $ot/$it.txt

##
##
it=IT011
echo "==> $it: feature: retry"
$iq agent -f ./examples/07_retry/run.yml -o $ot/$it.txt

##
##
it=IT012
echo "==> $it: feature: mcp"
$iq agent -f ./examples/08_mcp/run.yml -o $ot/$it.txt

##
##
it=IT013
echo "==> $it: feature: templates"
$iq agent -f ./examples/09_templates/run.yml -o $ot/$it.txt examples/09_templates/content/doc.txt

##
##
it=IT014
echo "==> $it: feature: foreach with selector"
$iq agent -f ./examples/10_foreach_selector/run.yml -o $ot/$it.txt examples/10_foreach_selector/content/sample-data.json

##
##
it=IT015
echo "==> $it: feature: shell command"
echo "https://example.com/" | $iq agent -f ./examples/11_command/run.yml -o $ot/$it.txt

##
##
it=IT016
echo "==> $it: feature: oauth2"
$iq agent -f ./examples/12_oauth2/run.yml -o $ot/$it.txt 

##
##
it=IT017
echo "==> $it: feature: splitter with array"
$iq agent -f examples/13_arrays/run.yml --splitter paragraph --array -o $ot/$it.txt examples/13_arrays/content/doc.txt
