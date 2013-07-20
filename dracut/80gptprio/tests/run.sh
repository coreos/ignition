#!/bin/sh -e

. ./include.sh

for i in test-*.sh; do
	echo -e "RUNNING $i\n"
	cleanup_root
	sh -e $i
	echo -e "\n"
done

echo SUCCESS
exit 0
