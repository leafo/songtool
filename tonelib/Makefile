unpack: $(patsubst %.song, unpacked/%, $(wildcard *.song))

unpacked/%: %.song
	mkdir -p $@
	unzip -o $< -d $@
	sed -i 's/\r$$//' "$@/the_song.dat"

%.song:: unpacked/%
	cd $< && zip -r ../../$@ .
