#!/bin/sh
# What a new tigwx writes, the published 0.1.0 has to read and write on top of, and
# the new one has to read back: people on one project run different versions.
#
#	sh scripts/compat.sh <new tigwx> <tigwx 0.1.0>
set -eu

new=$1
old=$2
dir=$(mktemp -d)
trap 'rm -rf "$dir"' EXIT
cd "$dir"
export TIG_ACTOR=compat

# jsonfield <name>: a string field of the JSON on stdin.
jsonfield() { sed -n "s/.*\"$1\": \"\([^\"]*\)\".*/\1/p" | head -1; }

printf '# Shop\n\nCarts are kept.\n\nPrices come from the table.\n' > README.md
"$new" init >/dev/null
carts=$("$new" mark -json -by human -confirm -source chat "Carts are kept." | jsonfield intent)
prices=$("$new" mark -json -by human "Prices come from the table." | jsonfield intent)
test -n "$carts"
test -n "$prices"
"$new" relate -by human -kind parallel -m apart "$carts" "$prices" >/dev/null
sed 's/Carts are kept\./Carts are kept a day./' README.md > README.tmp
mv README.tmp README.md
"$new" sync -by human -m legal -source "ticket 12" >/dev/null
"$new" snap >/dev/null

# The old one reads all of it, and what it does not know gets in nobody's way.
"$old" check >/dev/null
"$old" show "$carts" | grep -q 'Carts are kept a day'
"$old" show "$carts" | grep -q 'before .*: legal'
"$old" set -i "$carts" | grep -q 'Carts are kept a day'

# It writes on top of it, and the new one reads that back with nothing lost.
"$old" confirm -by human -m seen "$prices" >/dev/null
"$new" check >/dev/null
"$new" log "$prices" | grep -q 'why     seen'
"$new" show "$carts" | grep -q 'relate parallel'
"$new" log "$carts" | grep -q 'source  ticket 12'
echo "compatible both ways"
