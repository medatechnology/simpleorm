#!/bin/bash
EMAIL=$1



# Generate ssh key if  you don't already have.
# then go to github --> settings --> ssh key and 
# add the [sshfile].pub there (by pasting it)
ssh-keygen -t ed25519 -C "${EMAIL}"

# Check if 
git remote -v
git remote set-url origin git@github.com:medatechnology/goutil.git
git remote -v