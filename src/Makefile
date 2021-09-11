# TODO: Probably a more golang idiomatic way to compile and build files but this will work for now.

all:
	go build process.go
	go build -o  analytics/safetyAnalyzer  analytics/safetyAnalyzer.go
	go build -o  analytics/expAnalyzer   analytics/expAnalyzer.go
	go build -o  analytics/serverStats   analytics/serverStats.go

clean:
	$(MAKE) -C src clean
