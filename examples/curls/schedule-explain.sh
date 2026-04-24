#!/usr/bin/env bash
curl -X POST -H "Content-Type: application/json" -d @examples/workload-intents/vm-x86.json http://localhost:9090/api/v1alpha1/schedule/explain