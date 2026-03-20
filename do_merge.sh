#!/bin/sh
cd /Users/alexandr-bezobrazov/Arkadia/incident-replay-sim
exec > merge_out.txt 2>&1
echo "=== git merge-base ==="
git merge-base HEAD origin/main
echo "=== git merge ==="
git merge origin/main
echo "=== merge result: $? ==="
echo "=== git status ==="
git status
