#!/usr/bin/python

import argparse
import sys
from base64 import b64decode

# Define variale arguments for the program
file_parser = argparse.ArgumentParser()

# Define variable arguments for files
file_parser.add_argument('-i', dest='inFile', action='store', required=True, help="The file you wish to read and base64 decode")
file_parser.add_argument('-o', dest='outFile', action='store', required=True, help="The file you wish to save results to")

if len(sys.argv) < 3:
    file_parser.print_help(sys.stderr)
    sys.exit(1)

f = file_parser.parse_args()

with open (f.inFile, "r") as doc:
    with open(f.outFile, "w+") as r:
        for line in doc:
            current = line.split(":")
            i = current[0]
            node = current[1]
            job = current[2]
            res = b64decode(current[3])
            res = res.decode("utf-8")
            r.write(i+":"+node+":"+job+":"+res+"\n")
    r.close()
doc.close()
