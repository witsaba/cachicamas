#!/bin/sh
# Minimal askpass helper. Echoes the token from $GIT_ASKPASS_TOKEN.
# SECURITY: this script contains NO token literal; the token
# arrives at runtime via the GIT_ASKPASS_TOKEN env var that the
# parent Go process sets ONLY on the child's exec.Cmd environment.
# See runner.go (newAskpassScript) for the per-pid temp-file
# creation, mode 0o700, and the cleanup defer.
case "$1" in
  Username*) echo "x-access-token" ;;
  Password*) echo "$GIT_ASKPASS_TOKEN" ;;
  *) echo "$GIT_ASKPASS_TOKEN" ;;
esac
