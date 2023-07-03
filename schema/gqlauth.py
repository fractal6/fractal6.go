#!/bin/python3

'''Graphql auth formating for Dgraph

Usage:
    gqlauth.py AUTH_DIR SCHEMA

'''

import os
import re
from docopt import docopt


def find_rules(schema:str) -> list:
    with open(schema) as f:
        contents = f.read()
    pattern = r'<<(.*?)>>'
    sub_strings = [{'position': match.start(), 'pattern': match.group()} for match in re.finditer(pattern, contents, re.DOTALL)]
    return sub_strings


def replace_rules(auth_dir:str, schema:str, sub_strings:list) -> str:
    with open(schema) as f:
        contents = f.read()
    new_contents = contents
    for pattern in set(x["pattern"] for x in sub_strings):
        file_path = auth_dir + "/" + pattern[2:-2] + ".gql"
        with open(file_path) as gql_file:
            gql_content = gql_file.read().strip()
            gql_content = remove_comments(gql_content)
            new_contents = new_contents.replace(pattern, gql_content)
    return new_contents

def remove_comments(s):
    l = []
    for x in s.split("\n"):
        a = re.sub(r"#.*$", "", x)
        if a.strip():
            l.append(a)

    return "\n".join(l)


if __name__ == '__main__':
    args = docopt(__doc__, version='0.0')

    auth_dir = args["AUTH_DIR"]
    schema = args["SCHEMA"]

    rules = find_rules(schema)
    new_schema = replace_rules(auth_dir, schema, rules)
    print(new_schema)
