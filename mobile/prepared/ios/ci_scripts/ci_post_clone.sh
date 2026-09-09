#!/bin/zsh
set -eu
export HOMEBREW_NO_AUTO_UPDATE=1
brew install node@24
export PATH="$(brew --prefix node@24)/bin:$PATH"
cd "$CI_PRIMARY_REPOSITORY_PATH/mobile/prepared"
node scripts/verify-store-preparation.mjs
npm ci --include=dev
printf 'export NODE_BINARY="%s"\n' "$(command -v node)" > ios/.xcode.env.local
cd ios
pod install --deployment
