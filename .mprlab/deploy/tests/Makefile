GATEWAY_RUNTIME ?= $(HOME)/.local/share/mprlab-gateway/active/runtime
override TEST_DIRECTORY := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

.PHONY: test
test:
	CI=1 MPRLAB_GATEWAY_EXECUTABLE="$(GATEWAY_RUNTIME)/bin/mprlab-gateway" \
		uv run --python 3.13 --with ansible-core==2.19.8 --with pytest==9.0.2 \
		python -B -m pytest -q -p no:cacheprovider "$(TEST_DIRECTORY)test_npm_visibility.py" </dev/null
