#!/bin/bash
if [ "$GIT_COMMIT" = "8c60858fd972182f218cd0729421db609453ecae" ]; then
  echo "feat: AI intent parser with DeepSeek LLM"
elif [ "$GIT_COMMIT" = "8cb710d78cbdb6f93cce3db9785a9b97ca62a2b7" ]; then
  echo "feat: Mantle config + backend scaffold + router setup"
else
  cat
fi
