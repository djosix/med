#!/bin/bash

{
    echo 'package internal'
    echo
    echo 'var Nonce = []byte{'
    for (( i = 0; i < 16; i++ )); do
        printf '\t0x%02X,\n' "$(( RANDOM % 256 ))"
    done
    echo '}'
} > internal/nonce.go
