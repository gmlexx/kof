#!/usr/bin/env python

import re
import sys

import requests

processed = set()

for line in sys.stdin:
    for metric in re.findall("[A-Za-z_:]+{.+?}", line):
        if metric in processed:
            continue
        r = requests.get(
            f"http://localhost:8082/api/v1/query?query={metric}&time=1753359589.721"
        )
        response = r.json()
        if response["status"] == "success":
            if not response["data"]["result"]:
                print(f"{metric}")
        else:
            print(f"Metric {metric}: FAIL")
        processed.add(metric)
