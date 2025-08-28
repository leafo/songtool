unpack: $(patsubst %.song, unpacked/%, $(wildcard *.song))

unpacked/%: %.song
	mkdir -p $@
	unzip -o $< -d $@

%.song:: unpacked/%
	cd $< && zip -r ../../$@ .
