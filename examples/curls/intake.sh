#!/usr/bin/env bash
curl -X POST -H "Content-Type: application/json" -d @/tmp/payload.json http://localhost:9090/api/v1alpha1/intake/envelope