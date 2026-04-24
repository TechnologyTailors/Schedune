#!/usr/bin/env bash
curl -X POST -H "Content-Type: application/json" -d @examples/launch-specs/cloudhypervisor-execute.json http://localhost:9090/api/v1alpha1/launch/execute