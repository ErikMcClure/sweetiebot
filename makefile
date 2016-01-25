.PHONY: all

all:
  go build -tags release
  
nightly: pullsrc all

pullsrc:
	git pull